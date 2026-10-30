package handler

import (
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gin-gonic/gin"

	"github.com/bluemir/monosketch/internal/server/backend/documents"
)

// Server documents. See ADR-0011.

// documentResponse leaves out the owner name. The viewer only needs to know whether it is
// theirs.
type documentResponse struct {
	Id         string               `json:"id"`
	Name       string               `json:"name"`
	Content    string               `json:"content"`
	Revision   int                  `json:"revision"`
	Visibility documents.Visibility `json:"visibility,omitempty"`
	IsOwned    bool                 `json:"isOwned"`
	IsMine     bool                 `json:"isMine"`
	CanEdit    bool                 `json:"canEdit"`
	UpdatedAt  time.Time            `json:"updatedAt"`
}

func newDocumentResponse(document *documents.Document, viewer string) documentResponse {
	return documentResponse{
		Id:         document.Id,
		Name:       document.Name,
		Content:    document.Content,
		Revision:   document.Revision,
		Visibility: document.Visibility,
		IsOwned:    document.Owner != "",
		IsMine:     document.Owner != "" && document.Owner == viewer,
		CanEdit:    documents.CanEdit(document, viewer),
		UpdatedAt:  document.UpdatedAt,
	}
}

// viewerName is the logged in user name, or empty for a guest.
func viewerName(c *gin.Context) string {
	user, err := me(c)
	if err != nil {
		return ""
	}
	return user.Name
}

// bindDocumentRequest reads a JSON body no larger than the document limit.
func bindDocumentRequest(c *gin.Context, request any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, backends(c).Documents.MaxRequestBytes())
	if err := c.ShouldBindJSON(request); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return documents.ErrTooLarge
		}
		return errors.Wrap(documents.ErrInvalidContent, err.Error())
	}
	return nil
}

// @Router /api/v1/documents [post]
func CreateDocument(c *gin.Context) error {
	request := struct {
		Content string `json:"content" binding:"required"`
	}{}
	if err := bindDocumentRequest(c, &request); err != nil {
		return err
	}
	viewer := viewerName(c)
	document, err := backends(c).Documents.Create(c.Request.Context(), viewer, request.Content, c.ClientIP())
	if err != nil {
		return err
	}
	c.JSON(http.StatusCreated, newDocumentResponse(document, viewer))
	return nil
}

// @Router /api/v1/documents/{id} [get]
func GetDocument(c *gin.Context) error {
	viewer := viewerName(c)
	document, err := backends(c).Documents.Get(c.Request.Context(), c.Param("id"), viewer)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, newDocumentResponse(document, viewer))
	return nil
}

// UpdateDocument answers 409 if revision is not the latest one.
//
// @Router /api/v1/documents/{id} [put]
func UpdateDocument(c *gin.Context) error {
	request := struct {
		Revision int    `json:"revision" binding:"required"`
		Content  string `json:"content"  binding:"required"`
	}{}
	if err := bindDocumentRequest(c, &request); err != nil {
		return err
	}
	viewer := viewerName(c)
	document, err := backends(c).Documents.Update(c.Request.Context(), c.Param("id"), viewer, request.Revision, request.Content)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, newDocumentResponse(document, viewer))
	return nil
}

// @Router /api/v1/documents/{id}/visibility [put]
func SetDocumentVisibility(c *gin.Context) error {
	request := struct {
		Visibility documents.Visibility `json:"visibility" binding:"required"`
	}{}
	if err := c.ShouldBindJSON(&request); err != nil {
		return errors.Wrap(documents.ErrInvalidVisibility, err.Error())
	}
	viewer := viewerName(c)
	document, err := backends(c).Documents.SetVisibility(c.Request.Context(), c.Param("id"), viewer, request.Visibility)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, newDocumentResponse(document, viewer))
	return nil
}

// @Router /api/v1/documents/{id} [delete]
func DeleteDocument(c *gin.Context) error {
	if err := backends(c).Documents.Delete(c.Request.Context(), c.Param("id"), viewerName(c)); err != nil {
		return err
	}
	c.Status(http.StatusNoContent)
	return nil
}

// ListMyDocuments returns the documents the logged in user owns.
//
// @Router /api/v1/documents [get]
func ListMyDocuments(c *gin.Context) error {
	user, err := me(c)
	if err != nil {
		return err
	}
	list, err := backends(c).Documents.ListOwned(c.Request.Context(), user.Name)
	if err != nil {
		return err
	}
	c.JSON(http.StatusOK, list)
	return nil
}
