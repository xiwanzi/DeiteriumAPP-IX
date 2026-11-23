package httpapi

import (
	"bytes"
	"database/sql"
	"errors"
	"image/png"
	"io"
	"net/http"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

const supplyEditorArtworkPath = "/internal/v1/supply-editor/artwork/"
const publicSupplyEditorArtworkPath = "/api/v1/supply/editor-artwork/"
const maxSupplyEditorPNG = 2 << 20

func (s *Server) registerSupplyEditorArtwork(mux *http.ServeMux) {
	mux.HandleFunc("HEAD "+supplyEditorArtworkPath+"{hash}", s.supplyEditorArtworkExists)
	mux.HandleFunc("PUT "+supplyEditorArtworkPath+"{hash}", s.supplyEditorArtworkUpload)
	mux.HandleFunc("GET "+publicSupplyEditorArtworkPath+"{hash}", s.supplyEditorArtworkImage)
}

func (s *Server) supplyEditorArtworkExists(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.supplyEditorNode(w, r); !ok {
		return
	}
	hash := r.PathValue("hash")
	if !supplyItemFingerprint.MatchString(hash) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, err := s.Store.SupplyEditorArtworkInfo(r.Context(), hash, false)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		w.WriteHeader(http.StatusNotFound)
	case err != nil:
		w.WriteHeader(http.StatusServiceUnavailable)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) supplyEditorArtworkUpload(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.supplyEditorNode(w, r); !ok {
		return
	}
	hash := r.PathValue("hash")
	picture, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxSupplyEditorPNG))
	if err != nil || !supplyItemFingerprint.MatchString(hash) || store.Digest(picture) != hash {
		failure(w, r, 400, "INVALID_SUPPLY_IMAGE", "卡面大小或摘要校验失败。")
		return
	}
	config, err := png.DecodeConfig(bytes.NewReader(picture))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 4096 || config.Height > 4096 || config.Width*config.Height > 8388608 {
		failure(w, r, 400, "INVALID_SUPPLY_IMAGE", "卡面必须是尺寸合规的 PNG。")
		return
	}
	if _, err := png.Decode(bytes.NewReader(picture)); err != nil {
		failure(w, r, 400, "INVALID_SUPPLY_IMAGE", "卡面 PNG 解码失败。")
		return
	}
	if err := s.Store.SaveSupplyEditorArtwork(r.Context(), hash, picture, config.Width, config.Height); err != nil {
		failError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) supplyEditorArtworkImage(w http.ResponseWriter, r *http.Request) {
	hash := r.PathValue("hash")
	if !supplyItemFingerprint.MatchString(hash) {
		http.NotFound(w, r)
		return
	}
	picture, err := s.Store.PublishedSupplyEditorPicture(r.Context(), hash)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		failError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("ETag", `"`+hash+`"`)
	http.ServeContent(w, r, hash+".png", time.Time{}, bytes.NewReader(picture))
}
