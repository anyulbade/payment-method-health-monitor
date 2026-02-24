package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SetupSwagger(router *gin.Engine) {
	router.GET("/swagger/doc.json", func(c *gin.Context) {
		c.File("docs/swagger.json")
	})

	router.GET("/swagger/*any", func(c *gin.Context) {
		if c.Param("any") == "/doc.json" || c.Param("any") == "doc.json" {
			c.File("docs/swagger.json")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerUIHTML))
	})
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Payment Method Health Monitor - API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: '/swagger/doc.json',
      dom_id: '#swagger-ui',
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: "BaseLayout"
    });
  </script>
</body>
</html>`
