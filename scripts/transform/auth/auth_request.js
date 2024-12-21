// Auth service request transformation
(function () {
  // Add custom headers
  request.headers["X-Service-Version"] = "1.0";
  request.headers["X-Request-ID"] = generateUUID();

  // Remove sensitive headers
  delete request.headers["Authorization"];

  // Modify request body if it exists
  if (request.body) {
    // Add client info
    request.body.clientInfo = {
      timestamp: new Date().toISOString(),
      userAgent: request.headers["User-Agent"],
    };

    // Remove sensitive fields
    delete request.body.password;

    // Log transformation
    log
      .Debug()
      .Str("path", request.path)
      .Interface("headers", request.headers)
      .Msg("Auth request transformed");
  }
})();

// Helper function to generate UUID
function generateUUID() {
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, function (c) {
    var r = (Math.random() * 16) | 0,
      v = c == "x" ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}
