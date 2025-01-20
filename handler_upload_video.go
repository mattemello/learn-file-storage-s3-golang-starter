package main

import (
	"crypto/rand"
	"encoding/base32"
	"io"
	"mime"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"

	"fmt"
	"net/http"
)

var nameBucket = "tubely-2316120521"

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
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

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "You don't have the permision to take this video", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	_ = http.MaxBytesReader(w, r.Body, (1 << 30))

	mult, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusNotFound, "the file was not founded", err)
		return
	}
	defer mult.Close()
	contType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Not able to take the media type", err)
		return
	}
	if contType != "video/mp4" {
		respondWithError(w, http.StatusUnsupportedMediaType, "media type not supported", err)
		return
	}

	fil, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Can't create the file in the system", err)
		return
	}
	defer os.Remove("tubely-upload.mp4")
	defer fil.Close()

	_, err = io.Copy(fil, mult)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Can't copy the file", err)
		return
	}

	_, err = fil.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Can't reset the file pointer", err)
		return
	}

	var nameVide = make([]byte, 20)

	_, err = rand.Read(nameVide)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Can't generate a rand value", err)
		return
	}

	nameVideo := base32.HexEncoding.EncodeToString(nameVide)

	nameVideo += ".mp4"

	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{Bucket: &nameBucket, Key: &nameVideo, Body: fil, ContentType: &contType})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "It can't be saved", err)
		return
	}

	urlVideo := fmt.Sprintf("https://%s.s3.eu-north-1.amazonaws.com/%s", nameBucket, nameVideo)

	video.VideoURL = &urlVideo
	cfg.db.UpdateVideo(video)

	respondWithJSON(w, http.StatusOK, video)
}
