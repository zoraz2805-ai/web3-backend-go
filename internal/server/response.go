package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type responseWrapper struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

type pageResponseWrapper struct {
	Page  int  `json:"page"`
	Size  int  `json:"size"`
	Total *int `json:"total"`
	List  any  `json:"list"`
}

func writeSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, responseWrapper{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

func writePageSuccess(c *gin.Context, page int, size int, total *int, list any) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 30
	}

	writeSuccess(c, pageResponseWrapper{
		Page:  page,
		Size:  size,
		Total: total,
		List:  list,
	})
}

func writeError(c *gin.Context, httpStatus int, msg string) {
	c.JSON(httpStatus, responseWrapper{
		Code: httpStatus,
		Msg:  msg,
		Data: nil,
	})
}
