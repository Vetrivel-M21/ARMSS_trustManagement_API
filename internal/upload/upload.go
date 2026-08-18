package upload

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"trust-management/backend/internal/shared"

	"github.com/gin-gonic/gin"
)

const (
	uploadDir   = "./uploads"
	maxFileSize = 10 << 20 // 10 MB
)

var allowedExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".pdf": true,
}

type UploadHandler struct{}

func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

// UploadFile accepts a single multipart "file" field, validates its
// extension/size, and saves it under a random name to avoid path traversal
// and filename collisions. Returns a path servable via the static /uploads
// route registered in main.go.
func (h *UploadHandler) UploadFile(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		shared.SendAppError(c, http.StatusBadRequest, "No file provided")
		return
	}

	if fileHeader.Size > maxFileSize {
		shared.SendAppError(c, http.StatusBadRequest, "File exceeds the 10 MB size limit")
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedExtensions[ext] {
		shared.SendAppError(c, http.StatusBadRequest, "Unsupported file type. Allowed: jpg, jpeg, png, webp, pdf")
		return
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to prepare upload directory")
		return
	}

	randBytes := make([]byte, 16)
	if _, err := rand.Read(randBytes); err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to generate file name")
		return
	}
	filename := hex.EncodeToString(randBytes) + ext
	destPath := filepath.Join(uploadDir, filename)

	if err := c.SaveUploadedFile(fileHeader, destPath); err != nil {
		shared.SendAppError(c, http.StatusInternalServerError, "Failed to save uploaded file")
		return
	}

	shared.SendSuccess(c, http.StatusCreated, gin.H{
		"path": fmt.Sprintf("/uploads/%s", filename),
	})
}
