// Auth service request transformation
function transform(request) {
  // Add custom headers
  request.headers["X-Request-ID"] = generateUUID();
  request.headers["X-Service"] = "auth";

  // Remove sensitive headers
  delete request.headers["Authorization"];
  delete request.headers["Cookie"];

  // Transform request body if exists
  if (request.body) {
    let body = JSON.parse(request.body);

    // Add timestamp
    body.timestamp = new Date().toISOString();

    // Mask sensitive data
    if (body.password) {
      body.password = "********";
    }

    request.body = JSON.stringify(body);
  }

  return request;
}

// Helper function to generate UUID
function generateUUID() {
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, function (c) {
    var r = (Math.random() * 16) | 0,
      v = c == "x" ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}
