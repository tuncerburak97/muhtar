// Auth service response transformation
function transform(response) {
  // Add security headers
  response.headers["X-Content-Type-Options"] = "nosniff";
  response.headers["X-Frame-Options"] = "DENY";
  response.headers["X-XSS-Protection"] = "1; mode=block";

  // Remove internal headers
  delete response.headers["Server"];
  delete response.headers["X-Powered-By"];

  // Transform response body if exists
  if (response.body) {
    let body = JSON.parse(response.body);

    // Add response metadata
    body.responseTime = new Date().toISOString();

    // Remove sensitive data
    if (body.token) {
      body.token = body.token.substring(0, 10) + "...";
    }

    response.body = JSON.stringify(body);
  }

  return response;
}
