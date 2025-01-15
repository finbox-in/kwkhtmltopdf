package routes

import (
	"github.com/finbox-in/http/server"
	"github.com/gin-gonic/gin"
)

func AddHTMLToPDFRoutesV1(router *gin.RouterGroup, s *server.ServerHandler) {
	pdfRouteV1 := router.Group("/pdf")

	pdfRouteV1.POST("/", s.ConvertHTMLToPDF)

}
