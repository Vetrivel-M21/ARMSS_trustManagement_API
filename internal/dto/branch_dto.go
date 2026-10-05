package dto

import "time"

type CreateBranchRequest struct {
	BranchCode          string     `json:"branch_code" binding:"required"`
	Name                string     `json:"name" binding:"required"`
	TamilName           string     `json:"tamil_name"`
	LicenseNumber       string     `json:"license_number"`
	LicenseIssueDate    *time.Time `json:"license_issue_date"`
	LicenseExpiryDate   *time.Time `json:"license_expiry_date"`
	RegistrationDetails string     `json:"registration_details"`
	InchargeName        string     `json:"incharge_name"`
	Phone               string     `json:"phone"`
	Email               string     `json:"email"`
	AddressLine         string     `json:"address_line"`
	City                string     `json:"city"`
	State               string     `json:"state"`
	Pincode             string     `json:"pincode"`
	UPIID               string     `json:"upi_id"`
	QRCodePath          string     `json:"qr_code_path"`
	LogoPath            string     `json:"logo_path"`
}

type UpdateBranchRequest struct {
	Name                string     `json:"name"`
	TamilName           string     `json:"tamil_name"`
	LicenseNumber       string     `json:"license_number"`
	LicenseIssueDate    *time.Time `json:"license_issue_date"`
	LicenseExpiryDate   *time.Time `json:"license_expiry_date"`
	RegistrationDetails string     `json:"registration_details"`
	InchargeName        string     `json:"incharge_name"`
	Phone               string     `json:"phone"`
	Email               string     `json:"email"`
	AddressLine         string     `json:"address_line"`
	City                string     `json:"city"`
	State               string     `json:"state"`
	Pincode             string     `json:"pincode"`
	UPIID               string     `json:"upi_id"`
	QRCodePath          string     `json:"qr_code_path"`
	LogoPath            string     `json:"logo_path"`
	IsActive            *bool      `json:"is_active"`
}
