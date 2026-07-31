package controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/brian-l-johnson/CashierStatusBoard/v2/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NoteController struct{}

// maxUploadBytes caps an image upload. Big enough for a photo off a phone,
// small enough that a stuck upload can't fill the data volume.
const maxUploadBytes = 5 << 20

// allowedImageTypes maps a sniffed content type to the extension we will give
// the stored file. The extension comes from this table, never from the client.
var allowedImageTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// UploadDir is where uploaded images are written and served from. Set once at
// startup by the router, which also creates the directory.
var UploadDir = "./uploads"

// noteRequest is the write shape for a note. Active is a pointer so an update
// that omits it leaves the existing value alone rather than silently
// deactivating the note.
type noteRequest struct {
	Message  string `json:"message"`
	ImageURL string `json:"image_url"`
	Position int    `json:"position"`
	Active   *bool  `json:"active"`
	DwellSec int    `json:"dwell_sec"`
}

// apply validates the request and copies it onto a note. It returns a message
// suitable for handing back to the caller when validation fails.
func (r *noteRequest) apply(note *models.Note) (string, bool) {
	message := strings.TrimSpace(r.Message)
	if !models.ValidMessage(message) {
		return "message is required and must be 500 characters or fewer", false
	}
	// The whole defense against pointing a kiosk at an arbitrary host. An
	// operator-supplied external URL is never acceptable here, however
	// convenient it would be.
	if !models.ValidImageURL(r.ImageURL) {
		return "image_url must be empty or a path returned by /notes/upload", false
	}

	note.Message = message
	note.ImageURL = r.ImageURL
	note.Position = r.Position
	note.DwellSec = models.ClampDwellSec(r.DwellSec)
	if r.Active != nil {
		note.Active = *r.Active
	}
	return "", true
}

// get notes godoc
//
// @Summary Get active notes for the info board
// @Description Active notes ordered for display. Unauthenticated: display Pis have no session.
// @Tags Notes
// @Accept json
// @Produces json
// @Success 200 {array} models.Note
// @Success 304 {string} result
// @Router /notes [get]
func (h NoteController) GetNotes(c *gin.Context) {
	db := models.GetDB()
	// Non-nil so an empty board serializes as [] rather than null.
	notes := make([]models.Note, 0)

	// Tie-break on id: positions are not unique, and an unstable order would
	// make the ETag flap between two otherwise identical polls.
	result := db.Where("active = ?", true).Order("position asc, id asc").Find(&notes)
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": result.Error.Error()})
		return
	}

	// Marshal once and write these exact bytes. IndentedJSON would re-serialize
	// with different whitespace than what we hashed, and the ETag would never
	// match the body.
	body, err := json.Marshal(notes)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "failed to encode notes"})
		return
	}

	sum := sha256.Sum256(body)
	etag := `"` + hex.EncodeToString(sum[:]) + `"`
	c.Header("ETag", etag)
	c.Header("Cache-Control", "no-cache")

	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", body)
}

// get all notes godoc
//
// @Summary Get all notes including inactive ones
// @Tags Notes
// @Accept json
// @Produces json
// @Success 200 {array} models.Note
// @Router /notes/all [get]
func (h NoteController) GetAllNotes(c *gin.Context) {
	db := models.GetDB()
	notes := make([]models.Note, 0)

	result := db.Order("position asc, id asc").Find(&notes)
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": result.Error.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, notes)
}

// create note godoc
//
// @Summary Create a note
// @Tags Notes
// @Accept json
// @Produces json
// @Param note body models.Note true "note data"
// @Success 200 {string} result
// @Router /notes [post]
func (h NoteController) CreateNote(c *gin.Context) {
	var req noteRequest
	if err := c.BindJSON(&req); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": "failed to bind request"})
		return
	}

	// A note created without an explicit active flag should show up.
	note := models.Note{Active: true}
	if msg, ok := req.apply(&note); !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": msg})
		return
	}

	db := models.GetDB()
	result := db.Create(&note)
	if result.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": result.Error.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"status": "success", "message": "note created", "id": note.ID})
}

// update note godoc
//
// @Summary Update a note
// @Tags Notes
// @Accept json
// @Produces json
// @Param nid path string true "Note ID"
// @Param note body models.Note true "note data"
// @Success 200 {string} result
// @Router /notes/{nid} [put]
func (h NoteController) UpdateNote(c *gin.Context) {
	db := models.GetDB()
	var note models.Note

	result := db.First(&note, "id=?", c.Param("nid"))
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.IndentedJSON(http.StatusNotFound, gin.H{"status": "error", "message": "note not found"})
			return
		}
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": result.Error.Error()})
		return
	}

	var req noteRequest
	if err := c.BindJSON(&req); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": "failed to bind request"})
		return
	}
	if msg, ok := req.apply(&note); !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": msg})
		return
	}

	if save := db.Save(&note); save.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": save.Error.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"status": "success", "message": "note updated"})
}

// delete note godoc
//
// @Summary Delete a note
// @Description Soft deletes the note. Any uploaded image is intentionally left on disk.
// @Tags Notes
// @Accept json
// @Produces json
// @Param nid path string true "Note ID"
// @Success 200 {string} result
// @Router /notes/{nid} [delete]
func (h NoteController) DeleteNote(c *gin.Context) {
	db := models.GetDB()
	var note models.Note

	result := db.First(&note, "id=?", c.Param("nid"))
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.IndentedJSON(http.StatusNotFound, gin.H{"status": "error", "message": "note not found"})
			return
		}
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": result.Error.Error()})
		return
	}

	if del := db.Delete(&note); del.Error != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": del.Error.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{"status": "success", "message": "note deleted"})
}

// upload note image godoc
//
// @Summary Upload an image for a note
// @Description Returns the /uploads path to store on a note. The stored filename and extension are generated server side from the sniffed content type.
// @Tags Notes
// @Accept mpfd
// @Produces json
// @Param image formData file true "image file"
// @Success 200 {string} result
// @Router /notes/upload [post]
func (h NoteController) UploadImage(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes)

	fileHeader, err := c.FormFile("image")
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": "expected an 'image' file field of 5MB or less"})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": "could not read uploaded file"})
		return
	}
	defer src.Close()

	// Sniff the actual bytes. The client's Content-Type header and the
	// uploaded filename are both attacker-controlled and are never consulted.
	head := make([]byte, 512)
	n, err := io.ReadFull(src, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": "could not read uploaded file"})
		return
	}
	ext, ok := allowedImageTypes[http.DetectContentType(head[:n])]
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": "file is not a png, jpeg, gif, or webp image"})
		return
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "could not rewind uploaded file"})
		return
	}

	// Build the name ourselves; a user-supplied filename is never joined into
	// a path. The charset here is what lets models.ValidImageURL stay strict.
	random, err := models.GenerateRandomString(16)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "could not generate a filename"})
		return
	}
	name := random + ext

	dst, err := os.Create(filepath.Join(UploadDir, name))
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "could not write upload"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(dst.Name())
		c.IndentedJSON(http.StatusBadRequest, gin.H{"status": "error", "message": "upload failed or exceeded 5MB"})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{"status": "success", "image_url": "/uploads/" + name})
}
