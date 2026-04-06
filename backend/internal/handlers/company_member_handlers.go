package handlers

import (
	"github.com/google/uuid"

	"fmt"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

// GetCompanyMembers returns all members of a company
func GetCompanyMembers(c *fiber.Ctx) error {
	companyID := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company id"})
	}

	var members []models.CompanyMember
	database.DB.Preload("User").
		Where("company_id = ?", companyID).
		Find(&members)

	return c.JSON(members)
}

// InviteMemberByEmail sends an invitation to join a company
func InviteMemberByEmail(c *fiber.Ctx) error {
	companyID := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company id"})
	}

	inviterID := c.Locals("user_id").(uuid.UUID)

	type Input struct {
		Email string `json:"email"`
	}
	var input Input
	if err := c.BodyParser(&input); err != nil || input.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email is required"})
	}

	// 1. Check user exists
	var invitee models.User
	if err := database.DB.Where("email = ?", input.Email).First(&invitee).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User with this email was not found in the system"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error"})
	}

	// 2. Check if inviting self
	if invitee.ID == inviterID {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "You cannot invite yourself to a company"})
	}

	// 3. Check if already a member
	var existingMember models.CompanyMember
	if err := database.DB.Where("company_id = ? AND user_id = ?", companyID, invitee.ID).First(&existingMember).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "This user is already a member of this company"})
	}

	// 3. Check for pending invitation
	// 4. Get Company info for the notification
	var company models.Company
	database.DB.First(&company, "id = ?", companyID)

	companyUUID, _ := uuid.Parse(companyID)

	// 5. Create invitation
	invitation := models.CompanyInvitation{
		CompanyID: companyUUID,
		InviterID: inviterID,
		InviteeID: invitee.ID,
		Status:    "pending",
	}
	if err := database.DB.Create(&invitation).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create invitation"})
	}

	// 6. Create notification for invitee
	notification := models.DashboardNotification{
		UserID:       invitee.ID,
		InvitationID: &invitation.ID,
		Type:         "company_invite",
		Title:        "Company Invitation",
		Message:      fmt.Sprintf("You have been invited to join company '%s'", company.Name),
	}
	database.DB.Create(&notification)

	// 7. Create notification for inviter
	inviterNotification := models.DashboardNotification{
		UserID:  inviterID,
		Type:    "info",
		Title:   "Invitation Sent",
		Message: fmt.Sprintf("Invitation successfully sent to %s for company '%s'", input.Email, company.Name),
	}
	database.DB.Create(&inviterNotification)

	return c.Status(fiber.StatusCreated).JSON(invitation)
}

// AcceptCompanyInvitation adds user to company after they accept the invite
func AcceptCompanyInvitation(c *fiber.Ctx) error {
	invitationID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)

	var invitation models.CompanyInvitation
	if err := database.DB.Preload("Company").Where("id = ? AND invitee_id = ?", invitationID, userID).First(&invitation).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "invitation not found"})
	}

	if invitation.Status != "pending" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invitation is already " + invitation.Status})
	}

	// Start transaction
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Update all pending invitations for this company/user to 'accepted'
		if err := tx.Model(&models.CompanyInvitation{}).Where("company_id = ? AND invitee_id = ? AND status = 'pending'", invitation.CompanyID, invitation.InviteeID).Update("status", "accepted").Error; err != nil {
			return err
		}

		// 2. Add as company member
		member := models.CompanyMember{
			CompanyID: invitation.CompanyID,
			UserID:    invitation.InviteeID,
			Role:      "member",
		}
		if err := tx.Create(&member).Error; err != nil {
			return err
		}

		// 3. Mark all related invite notifications as read
		tx.Model(&models.DashboardNotification{}).Where("user_id = ? AND type = 'company_invite' AND invitation_id IN (SELECT id FROM company_invitations WHERE company_id = ? AND invitee_id = ?)", userID, invitation.CompanyID, userID).Update("is_read", true)

		// 4. Notify inviter
		var invitee models.User
		tx.First(&invitee, "id = ?", userID)
		notification := models.DashboardNotification{
			UserID:  invitation.InviterID,
			Type:    "info",
			Title:   "Invitation Accepted",
			Message: fmt.Sprintf("%s has accepted your invitation to join '%s'", invitee.Name, invitation.Company.Name),
		}
		if err := tx.Create(&notification).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to accept invitation"})
	}

	return c.JSON(fiber.Map{"message": "Invitation accepted successfully"})
}

// DeclineCompanyInvitation updates status to declined
func DeclineCompanyInvitation(c *fiber.Ctx) error {
	invitationID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)

	var invitation models.CompanyInvitation
	if err := database.DB.Preload("Company").Where("id = ? AND invitee_id = ?", invitationID, userID).First(&invitation).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "invitation not found"})
	}

	if invitation.Status != "pending" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invitation is already " + invitation.Status})
	}

	// Start transaction
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Update all pending invitations for this company/user to 'declined'
		if err := tx.Model(&models.CompanyInvitation{}).Where("company_id = ? AND invitee_id = ? AND status = 'pending'", invitation.CompanyID, invitation.InviteeID).Update("status", "declined").Error; err != nil {
			return err
		}

		// 2. Mark related notifications as read
		tx.Model(&models.DashboardNotification{}).Where("user_id = ? AND type = 'company_invite' AND invitation_id IN (SELECT id FROM company_invitations WHERE company_id = ? AND invitee_id = ?)", userID, invitation.CompanyID, userID).Update("is_read", true)

		// 3. Notify inviter
		var invitee models.User
		tx.First(&invitee, "id = ?", userID)
		notification := models.DashboardNotification{
			UserID:  invitation.InviterID,
			Type:    "info",
			Title:   "Invitation Declined",
			Message: fmt.Sprintf("%s has declined your invitation to join '%s'", invitee.Name, invitation.Company.Name),
		}
		if err := tx.Create(&notification).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to decline invitation"})
	}

	return c.JSON(fiber.Map{"message": "Invitation declined"})
}

func RemoveCompanyMember(c *fiber.Ctx) error {
	companyID := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company id"})
	}
	memberID := c.Params("memberId")
	if memberID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid member id"})
	}

	result := database.DB.Where("id = ? AND company_id = ?", memberID, companyID).Delete(&models.CompanyMember{})
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "member not found"})
	}
	return c.JSON(fiber.Map{"message": "member removed"})
}
