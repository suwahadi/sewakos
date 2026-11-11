package owner

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/suwahadi/sewakos/backend/internal/authctx"
	"github.com/suwahadi/sewakos/backend/internal/media"
	"github.com/suwahadi/sewakos/backend/internal/money"
	"github.com/suwahadi/sewakos/backend/internal/platform/postgres/dbgen"
	"github.com/suwahadi/sewakos/backend/internal/platform/storage"
	"github.com/suwahadi/sewakos/backend/internal/response"
	"github.com/suwahadi/sewakos/backend/internal/validation"
)

const (
	maxPhotosPerProperty = 30
	presignExpiry        = 10 * time.Minute
	processTimeout       = 30 * time.Second
	photoCacheControl    = "public, max-age=31536000, immutable"

	msgPhotoInvalid       = "File foto tidak ditemukan atau tidak valid."
	msgPhotoUnprocessable = "File foto tidak dapat diproses."
	msgPhotoLimit         = "Jumlah foto sudah mencapai batas."
)

var uploadExtensions = map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp"}

var errPhotoLimit = errors.New("photo limit reached")

func PresignHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body := decodeBody(r)
		v := validation.New()
		v.Field("content_type", body["content_type"]).Required().String().In("image/jpeg", "image/png", "image/webp")
		v.Field("size", body["size"]).Required().Integer().BetweenNumber(1, float64(deps.UploadMaxBytes))
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		contentType := body["content_type"].(string)
		key := fmt.Sprintf("uploads/%d/%s.%s", user.UserID, uuid.NewString(), uploadExtensions[contentType])
		expiresAt := time.Now().Add(presignExpiry)
		url, err := deps.Storage.PresignPut(r.Context(), key, contentType, presignExpiry)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		response.Success(w, http.StatusOK, "", map[string]any{
			"upload": map[string]any{
				"method": http.MethodPut, "url": url, "headers": map[string]string{"Content-Type": contentType},
				"object_key": key, "expires_at": money.FormatExpiresAt(expiresAt),
			},
		})
	}
}

func AddPhotoHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		propertyID, ok := parseID(r, "property")
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()
		owned, err := deps.Store.OwnerPropertyExists(ctx, dbgen.OwnerPropertyExistsParams{ID: propertyID, OwnerID: user.UserID})
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if !owned {
			response.NotFoundHandler(w, r)
			return
		}

		body := decodeBody(r)
		v := validation.New()
		v.Field("object_key", body["object_key"]).Required().String()
		roomField := v.Field("room_id", body["room_id"]).Optional().Integer()
		roomID, hasRoom := wholeNumber(body["room_id"])
		if hasRoom {
			belongs, err := deps.Store.RoomBelongsToProperty(ctx, dbgen.RoomBelongsToPropertyParams{ID: roomID, PropertyID: propertyID})
			if err != nil {
				response.InternalErrorHandler(w, r)
				return
			}
			roomField.Exists(belongs)
		}
		v.Field("caption", body["caption"]).Optional().String().MaxStringLen(255)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		uploadKey := body["object_key"].(string)
		if !strings.HasPrefix(uploadKey, fmt.Sprintf("uploads/%d/", user.UserID)) {
			response.Fail(w, http.StatusBadRequest, msgPhotoInvalid, nil)
			return
		}
		count, err := deps.Store.CountPropertyPhotos(ctx, propertyID)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if count >= maxPhotosPerProperty {
			response.Fail(w, http.StatusBadRequest, msgPhotoLimit, nil)
			return
		}

		original, status := fetchUpload(ctx, deps, uploadKey)
		if status != 0 {
			failWithStatus(w, r, status, msgPhotoInvalid)
			return
		}
		processCtx, cancel := context.WithTimeout(ctx, processTimeout)
		defer cancel()
		image, err := deps.Images.Process(processCtx, original)
		if errors.Is(err, media.ErrUnprocessable) {
			response.Fail(w, http.StatusBadRequest, msgPhotoUnprocessable, nil)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		keyBase := fmt.Sprintf("properties/%d/%s", propertyID, uuid.NewString())
		largeKey, thumbKey := keyBase+".webp", keyBase+"_thumb.webp"
		if err := uploadWebp(ctx, deps, largeKey, image.Large, thumbKey, image.Thumb); err != nil {
			queueStorageDeletion(ctx, deps.Store, largeKey, thumbKey)
			response.InternalErrorHandler(w, r)
			return
		}

		photo := dbgen.InsertPropertyPhotoParams{
			PropertyID: propertyID, RoomID: pgtype.Int8{Int64: roomID, Valid: hasRoom},
			ObjectKey: largeKey, ThumbKey: thumbKey, Width: int32(image.Width), Height: int32(image.Height),
			SizeBytes: int32(len(image.Large)), Caption: optionalText(body["caption"]),
		}
		inserted, err := insertPhoto(ctx, deps, r, user, photo)
		if err != nil {
			queueStorageDeletion(ctx, deps.Store, largeKey, thumbKey)
			if errors.Is(err, errPhotoLimit) {
				response.Fail(w, http.StatusBadRequest, msgPhotoLimit, nil)
				return
			}
			response.InternalErrorHandler(w, r)
			return
		}
		if err := deps.Storage.Remove(ctx, uploadKey); err != nil {
			queueStorageDeletion(ctx, deps.Store, uploadKey)
		}

		response.Success(w, http.StatusOK, "Foto ditambahkan.", map[string]any{
			"photo": map[string]any{
				"id": inserted.ID, "url": photoURL(deps.StorageBaseURL, largeKey),
				"thumbnail_url": photoURL(deps.StorageBaseURL, thumbKey), "caption": nullableText(photo.Caption),
				"room_id": nullableInt8(photo.RoomID), "sort_order": inserted.SortOrder,
			},
		})
	}
}

func fetchUpload(ctx context.Context, deps Deps, key string) ([]byte, int) {
	size, err := deps.Storage.Size(ctx, key)
	if errors.Is(err, storage.ErrNotFound) || (err == nil && size > deps.UploadMaxBytes) {
		return nil, http.StatusBadRequest
	}
	if err != nil {
		return nil, http.StatusInternalServerError
	}
	data, err := deps.Storage.Get(ctx, key)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, http.StatusBadRequest
	}
	if err != nil {
		return nil, http.StatusInternalServerError
	}
	return data, 0
}

func failWithStatus(w http.ResponseWriter, r *http.Request, status int, message string) {
	if status == http.StatusInternalServerError {
		response.InternalErrorHandler(w, r)
		return
	}
	response.Fail(w, status, message, nil)
}

func uploadWebp(ctx context.Context, deps Deps, largeKey string, large []byte, thumbKey string, thumb []byte) error {
	if err := deps.Storage.Put(ctx, largeKey, large, "image/webp", photoCacheControl); err != nil {
		return err
	}
	return deps.Storage.Put(ctx, thumbKey, thumb, "image/webp", photoCacheControl)
}

func insertPhoto(ctx context.Context, deps Deps, r *http.Request, user authctx.User, photo dbgen.InsertPropertyPhotoParams) (dbgen.InsertPropertyPhotoRow, error) {
	tx, err := deps.DB.Begin(ctx)
	if err != nil {
		return dbgen.InsertPropertyPhotoRow{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := dbgen.New(tx)

	if _, err := queries.LockOwnerProperty(ctx, dbgen.LockOwnerPropertyParams{ID: photo.PropertyID, OwnerID: user.UserID}); err != nil {
		return dbgen.InsertPropertyPhotoRow{}, fmt.Errorf("lock property %d: %w", photo.PropertyID, err)
	}
	count, err := queries.CountPropertyPhotos(ctx, photo.PropertyID)
	if err != nil {
		return dbgen.InsertPropertyPhotoRow{}, err
	}
	if count >= maxPhotosPerProperty {
		return dbgen.InsertPropertyPhotoRow{}, errPhotoLimit
	}
	inserted, err := queries.InsertPropertyPhoto(ctx, photo)
	if err != nil {
		return dbgen.InsertPropertyPhotoRow{}, fmt.Errorf("insert photo: %w", err)
	}
	after := map[string]any{
		"property_id": photo.PropertyID, "room_id": nullableInt8(photo.RoomID), "object_key": photo.ObjectKey,
		"caption": nullableText(photo.Caption), "sort_order": inserted.SortOrder,
	}
	if err := recordAudit(ctx, queries, r, user, "photo.create", "property_photo", inserted.ID, nil, after); err != nil {
		return dbgen.InsertPropertyPhotoRow{}, err
	}
	return inserted, tx.Commit(ctx)
}

func UpdatePhotoHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		photoID, ok := parseID(r, "photo")
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()
		tx, err := deps.DB.Begin(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()
		queries := dbgen.New(tx)

		photo, err := queries.LockOwnerPhoto(ctx, dbgen.LockOwnerPhotoParams{ID: photoID, OwnerID: user.UserID})
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}

		body := decodeBody(r)
		v := validation.New()
		v.Field("caption", body["caption"]).Optional().String().MaxStringLen(255)
		v.Field("sort_order", body["sort_order"]).Optional().Integer().MinNumber(0)
		if !v.Valid() {
			response.Fail(w, http.StatusUnprocessableEntity, response.MsgValidationFailed, v.Errors())
			return
		}

		update := dbgen.UpdatePropertyPhotoParams{ID: photo.ID, Caption: photo.Caption, SortOrder: photo.SortOrder}
		if _, sent := body["caption"]; sent {
			update.Caption = optionalText(body["caption"])
		}
		if sortOrder, ok := wholeNumber(body["sort_order"]); ok {
			update.SortOrder = int32(sortOrder)
		}
		if err := queries.UpdatePropertyPhoto(ctx, update); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		before := map[string]any{"caption": nullableText(photo.Caption), "sort_order": photo.SortOrder}
		after := map[string]any{"caption": nullableText(update.Caption), "sort_order": update.SortOrder}
		if err := recordAudit(ctx, queries, r, user, "photo.update", "property_photo", photo.ID, before, after); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "Foto diperbarui.", nil)
	}
}

func DeletePhotoHandler(deps Deps, user authctx.User) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		photoID, ok := parseID(r, "photo")
		if !ok {
			response.NotFoundHandler(w, r)
			return
		}
		ctx := r.Context()
		tx, err := deps.DB.Begin(ctx)
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()
		queries := dbgen.New(tx)

		photo, err := queries.LockOwnerPhoto(ctx, dbgen.LockOwnerPhotoParams{ID: photoID, OwnerID: user.UserID})
		if errors.Is(err, pgx.ErrNoRows) {
			response.NotFoundHandler(w, r)
			return
		}
		if err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := queries.DeletePropertyPhoto(ctx, photo.ID); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		before := map[string]any{
			"property_id": photo.PropertyID, "room_id": nullableInt8(photo.RoomID), "object_key": photo.ObjectKey,
			"caption": nullableText(photo.Caption), "sort_order": photo.SortOrder,
		}
		if err := recordAudit(ctx, queries, r, user, "photo.delete", "property_photo", photo.ID, before, nil); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			response.InternalErrorHandler(w, r)
			return
		}
		response.Success(w, http.StatusOK, "Foto dihapus.", nil)
	}
}
