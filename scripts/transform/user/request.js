// User service request transformation
function transform(request) {
  // Add custom headers
  request.headers["X-Request-ID"] = generateUUID();
  request.headers["X-Service"] = "user";

  // Transform request body if exists
  if (request.body) {
    let body = JSON.parse(request.body);

    // Add request metadata
    body.requestTime = new Date().toISOString();

    // Mask sensitive fields
    if (body.email) {
      const [name, domain] = body.email.split("@");
      body.email = name[0] + "***@" + domain;
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
