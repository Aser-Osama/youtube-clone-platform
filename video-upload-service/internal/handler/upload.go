package handler

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

func UploadVideo(c *gin.Context) {
    file, _ := c.FormFile("file")
    c.SaveUploadedFile(file, "/tmp/"+file.Filename)
    c.JSON(http.StatusOK, gin.H{"message": "uploaded successfully"})
}

