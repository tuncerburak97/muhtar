// Auth service response transformation
(function () {
  // Add security headers
  response.headers["Strict-Transport-Security"] =
    "max-age=31536000; includeSubDomains";
  response.headers["X-Content-Type-Options"] = "nosniff";
  response.headers["X-Frame-Options"] = "DENY";
  response.headers["X-XSS-Protection"] = "1; mode=block";

  // Remove internal headers
  delete response.headers["X-Powered-By"];
  delete response.headers["Server"];

  // Modify response body if it exists and is JSON
  if (response.body && typeof response.body === "object") {
    // Add metadata
    response.body.metadata = {
      timestamp: new Date().toISOString(),
      version: "1.0",
    };

    // Remove sensitive data
    if (response.body.user) {
      delete response.body.user.password;
      delete response.body.user.secretKey;
    }

    // Transform error responses
    if (response.statusCode >= 400) {
      response.body = {
        error: {
          code: response.statusCode,
          message: response.body.message || "An error occurred",
          timestamp: new Date().toISOString(),
        },
      };
    }

    // Log transformation
    log
      .Debug()
      .Int("statusCode", response.statusCode)
      .Interface("headers", response.headers)
      .Msg("Auth response transformed");
  }
})();
