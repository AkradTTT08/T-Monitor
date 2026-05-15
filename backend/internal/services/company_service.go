package services

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
	"gorm.io/gorm"
)

type CompanyService interface {
	GetCompaniesForUser(userID uuid.UUID) ([]models.Company, error)
	GetCompanyByID(companyID string, userID uuid.UUID) (*models.Company, error)
	CreateCompany(name, description string, userID uuid.UUID) (*models.Company, error)
	UpdateCompany(companyID string, name, description string, userID uuid.UUID) (*models.Company, error)
	DeleteCompany(companyID string, userID uuid.UUID) error
	
	GetCompanyMembers(companyID string) ([]models.CompanyMember, error)
	InviteMember(companyID string, email string, inviterID uuid.UUID) (*models.CompanyInvitation, error)
	AcceptInvitation(invitationID string, userID uuid.UUID) error
	DeclineInvitation(invitationID string, userID uuid.UUID) error
	RemoveMember(companyID, memberID string) error
}

type companyService struct {
	db *gorm.DB
}

func NewCompanyService(db *gorm.DB) CompanyService {
	if db == nil {
		db = database.DB // Use global instance if not explicitly passed (for tests we can pass it if we want)
	}
	return &companyService{db: db}
}

func (s *companyService) GetCompaniesForUser(userID uuid.UUID) ([]models.Company, error) {
	var companies []models.Company
	err := s.db.Preload("Projects").Preload("Owner").Preload("Members.User").
		Where("user_id = ? OR id IN (SELECT company_id FROM company_members WHERE user_id = ?)", userID, userID).
		Find(&companies).Error
	return companies, err
}

func (s *companyService) GetCompanyByID(companyID string, userID uuid.UUID) (*models.Company, error) {
	var company models.Company
	err := s.db.Preload("Projects").Preload("Owner").Preload("Members.User").
		Where("user_id = ? OR id IN (SELECT company_id FROM company_members WHERE user_id = ?)", userID, userID).
		First(&company, "id = ?", companyID).Error
	if err != nil {
		return nil, errors.New("Company not found or unauthorized")
	}
	return &company, nil
}

func (s *companyService) CreateCompany(name, description string, userID uuid.UUID) (*models.Company, error) {
	company := models.Company{
		Name:        name,
		Description: description,
		UserID:      userID,
	}

	if err := s.db.Create(&company).Error; err != nil {
		return nil, err
	}
	return &company, nil
}

func (s *companyService) UpdateCompany(companyID string, name, description string, userID uuid.UUID) (*models.Company, error) {
	var company models.Company
	if err := s.db.Where("user_id = ?", userID).First(&company, "id = ?", companyID).Error; err != nil {
		return nil, errors.New("Company not found or unauthorized")
	}

	updateData := map[string]interface{}{
		"name":        name,
		"description": description,
	}

	if err := s.db.Model(&company).Updates(updateData).Error; err != nil {
		return nil, err
	}
	return &company, nil
}

func (s *companyService) DeleteCompany(companyID string, userID uuid.UUID) error {
	var company models.Company
	if err := s.db.Where("user_id = ?", userID).First(&company, "id = ?", companyID).Error; err != nil {
		return errors.New("Company not found or unauthorized")
	}

	s.db.Where("company_id = ?", company.ID).Delete(&models.Project{})
	return s.db.Delete(&company).Error
}

func (s *companyService) GetCompanyMembers(companyID string) ([]models.CompanyMember, error) {
	var members []models.CompanyMember
	err := s.db.Preload("User").Where("company_id = ?", companyID).Find(&members).Error
	return members, err
}

func (s *companyService) InviteMember(companyID string, email string, inviterID uuid.UUID) (*models.CompanyInvitation, error) {
	var invitee models.User
	if err := s.db.Where("email = ?", email).First(&invitee).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("User with this email was not found in the system")
		}
		return nil, errors.New("database error")
	}

	if invitee.ID == inviterID {
		return nil, errors.New("You cannot invite yourself to a company")
	}

	var existingMember models.CompanyMember
	if err := s.db.Where("company_id = ? AND user_id = ?", companyID, invitee.ID).First(&existingMember).Error; err == nil {
		return nil, errors.New("This user is already a member of this company")
	}

	var company models.Company
	if err := s.db.First(&company, "id = ?", companyID).Error; err != nil {
		return nil, errors.New("Company not found")
	}

	companyUUID, _ := uuid.Parse(companyID)

	invitation := models.CompanyInvitation{
		CompanyID: companyUUID,
		InviterID: inviterID,
		InviteeID: invitee.ID,
		Status:    "pending",
	}
	if err := s.db.Create(&invitation).Error; err != nil {
		return nil, errors.New("failed to create invitation")
	}

	notification := models.DashboardNotification{
		UserID:       invitee.ID,
		InvitationID: &invitation.ID,
		Type:         "company_invite",
		Title:        "Company Invitation",
		Message:      fmt.Sprintf("You have been invited to join company '%s'", company.Name),
	}
	s.db.Create(&notification)

	inviterNotification := models.DashboardNotification{
		UserID:  inviterID,
		Type:    "info",
		Title:   "Invitation Sent",
		Message: fmt.Sprintf("Invitation successfully sent to %s for company '%s'", email, company.Name),
	}
	s.db.Create(&inviterNotification)

	return &invitation, nil
}

func (s *companyService) AcceptInvitation(invitationID string, userID uuid.UUID) error {
	var invitation models.CompanyInvitation
	if err := s.db.Preload("Company").Where("id = ? AND invitee_id = ?", invitationID, userID).First(&invitation).Error; err != nil {
		return errors.New("invitation not found")
	}

	if invitation.Status != "pending" {
		return errors.New("invitation is already " + invitation.Status)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.CompanyInvitation{}).Where("company_id = ? AND invitee_id = ? AND status = 'pending'", invitation.CompanyID, invitation.InviteeID).Update("status", "accepted").Error; err != nil {
			return err
		}

		member := models.CompanyMember{
			CompanyID: invitation.CompanyID,
			UserID:    invitation.InviteeID,
			Role:      "member",
		}
		if err := tx.Create(&member).Error; err != nil {
			return err
		}

		tx.Model(&models.DashboardNotification{}).Where("user_id = ? AND type = 'company_invite' AND invitation_id IN (SELECT id FROM company_invitations WHERE company_id = ? AND invitee_id = ?)", userID, invitation.CompanyID, userID).Update("is_read", true)

		var invitee models.User
		tx.First(&invitee, "id = ?", userID)
		notification := models.DashboardNotification{
			UserID:  invitation.InviterID,
			Type:    "info",
			Title:   "Invitation Accepted",
			Message: fmt.Sprintf("%s has accepted your invitation to join '%s'", invitee.Name, invitation.Company.Name),
		}
		return tx.Create(&notification).Error
	})
}

func (s *companyService) DeclineInvitation(invitationID string, userID uuid.UUID) error {
	var invitation models.CompanyInvitation
	if err := s.db.Preload("Company").Where("id = ? AND invitee_id = ?", invitationID, userID).First(&invitation).Error; err != nil {
		return errors.New("invitation not found")
	}

	if invitation.Status != "pending" {
		return errors.New("invitation is already " + invitation.Status)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.CompanyInvitation{}).Where("company_id = ? AND invitee_id = ? AND status = 'pending'", invitation.CompanyID, invitation.InviteeID).Update("status", "declined").Error; err != nil {
			return err
		}

		tx.Model(&models.DashboardNotification{}).Where("user_id = ? AND type = 'company_invite' AND invitation_id IN (SELECT id FROM company_invitations WHERE company_id = ? AND invitee_id = ?)", userID, invitation.CompanyID, userID).Update("is_read", true)

		var invitee models.User
		tx.First(&invitee, "id = ?", userID)
		notification := models.DashboardNotification{
			UserID:  invitation.InviterID,
			Type:    "info",
			Title:   "Invitation Declined",
			Message: fmt.Sprintf("%s has declined your invitation to join '%s'", invitee.Name, invitation.Company.Name),
		}
		return tx.Create(&notification).Error
	})
}

func (s *companyService) RemoveMember(companyID, memberID string) error {
	result := s.db.Where("id = ? AND company_id = ?", memberID, companyID).Delete(&models.CompanyMember{})
	if result.RowsAffected == 0 {
		return errors.New("member not found")
	}
	return nil
}
