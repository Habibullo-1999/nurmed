package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerSwaggerRoutes(router *gin.Engine) {
	router.GET("/swagger", swaggerUIHandler)
	router.GET("/swagger/", swaggerUIHandler)
	router.GET("/swagger/index.html", swaggerUIHandler)
	router.StaticFile("/swagger/openapi.yaml", "api/openapi/openapi.yaml")
}

func swaggerUIHandler(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
}

const swaggerHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>NurMed Swagger</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <style>
    html, body { margin: 0; padding: 0; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function() {
      window.ui = SwaggerUIBundle({
        url: '/swagger/openapi.yaml',
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis],
        layout: "BaseLayout"
      });
    };
  </script>
</body>
</html>`
