package handlers

import (
	"github.com/google/uuid"

	"github.com/gofiber/fiber/v2"

	"github.com/monitor-api/backend/internal/services"
)

// GetCompanyMembers returns all members of a company
func GetCompanyMembers(c *fiber.Ctx) error {
	companyID := c.Params("id")
	if companyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid company id"})
	}

	svc := services.NewCompanyService(nil)
	members, err := svc.GetCompanyMembers(companyID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

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

	svc := services.NewCompanyService(nil)
	invitation, err := svc.InviteMember(companyID, input.Email, inviterID)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "User with this email was not found in the system" {
			status = fiber.StatusNotFound
		} else if err.Error() == "You cannot invite yourself to a company" || err.Error() == "This user is already a member of this company" {
			status = fiber.StatusConflict
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(invitation)
}

// AcceptCompanyInvitation adds user to company after they accept the invite
func AcceptCompanyInvitation(c *fiber.Ctx) error {
	invitationID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)

	svc := services.NewCompanyService(nil)
	if err := svc.AcceptInvitation(invitationID, userID); err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "invitation not found" {
			status = fiber.StatusNotFound
		} else if err.Error() != "failed to accept invitation" {
			status = fiber.StatusBadRequest
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Invitation accepted successfully"})
}

// DeclineCompanyInvitation updates status to declined
func DeclineCompanyInvitation(c *fiber.Ctx) error {
	invitationID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)

	svc := services.NewCompanyService(nil)
	if err := svc.DeclineInvitation(invitationID, userID); err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "invitation not found" {
			status = fiber.StatusNotFound
		} else if err.Error() != "failed to decline invitation" {
			status = fiber.StatusBadRequest
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
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

	svc := services.NewCompanyService(nil)
	if err := svc.RemoveMember(companyID, memberID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "member removed"})
}
