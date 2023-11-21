package utils

const LocalServerNotOnline = `
HTTP/1.1 400 Bad Request
Content-Type: text/html; charset=utf-8

<!DOCTYPE html>
<html>
<head>
    <title>Localport</title>
</head>
<body>
    <h1>Your local server is not online</h1>
	<h2>Try again after starting the server</h2>
</body>
</html>`

const UnregisteredSubdomain = `
<!DOCTYPE html>
<html>
<head>
    <title>Localport</title>
</head>
<body>
    <h1>There is no tunnel running on this endpoint</h1>
</body>
</html>`
