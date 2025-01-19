package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here

	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error in the parser", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error in the parser", err)
		return
	}
	defer file.Close()

	/* readall, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Not possible read all", err)
		return
	} */

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "You don't have the permision to take this video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You don't have the permision to take this video", err)
		return
	}

	typeHead := header.Header.Get("Content-Type")
	if typeHead == "" {
		respondWithError(w, http.StatusBadRequest, "Missing content-type", err)
		return
	}
	mimetype, _, err := mime.ParseMediaType(typeHead)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Bad content-type", err)
		return
	}

	if mimetype != "image/jpeg" && mimetype != "image/png" {
		respondWithError(w, http.StatusUnsupportedMediaType, "Media type not supported", err)
		return
	}

	fileName := videoIDString + "." + strings.Split(typeHead, "/")[1]
	pathf := filepath.Join(cfg.assetsRoot, fileName)

	filef, err := os.Create(pathf)
	if err != nil {
		respondWithError(w, http.StatusConflict, "Missing content-type", err)
		return
	}

	_, err = io.Copy(filef, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Missing content-type", err)
		return
	}
	/* thumbnail64Base := base64.StdEncoding.EncodeToString(readall)
	var thum64media = fmt.Sprintf("data:%s;base64,%s", typeHead, thumbnail64Base) */

	var thumUrl = fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, fileName)

	video.ThumbnailURL = &thumUrl
	cfg.db.UpdateVideo(video)

	respondWithJSON(w, http.StatusOK, video)
}
