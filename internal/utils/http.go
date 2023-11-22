package utils

import "fmt"

func LocalServerNotOnline(endpoint string) string {
	return Trim(fmt.Sprintf(`
HTTP/1.1 400 Bad Request
Content-Type: text/html; charset=utf-8

<!DOCTYPE html>
<html lang="en">
    <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Localport</title>
    <style>
        body {
        padding: 0 24px;
        margin: 0;
        height: 80vh;
        display: flex;
        justify-content: center;
        align-items: center;
        font-family: "Inter";
        }
        #main {
        display: flex;
        flex-direction: column;
        align-items: center;
        }
        h1,
        h2 {
        font-weight: 300;
        }
    </style>
    </head>
    <body>
    <div id="main">
        <h1>Unable to establish connection with the local server</h1>
        <h2>Please ensure that the local server is running and then reload this page</h2>
        <svg xmlns="http://www.w3.org/2000/svg" width="400" height="200">
        <!-- Local Server -->
        <svg
            x="85"
            y="80"
            width="30"
            height="30"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
        >
            <rect width="20" height="8" x="2" y="14" rx="2" />
            <path d="M6.01 18H6" />
            <path d="M10.01 18H10" />
            <path d="M15 10v4" />
            <path d="M17.84 7.17a4 4 0 0 0-5.66 0" />
            <path d="M20.66 4.34a8 8 0 0 0-11.31 0" />
        </svg>
        <text x="100" y="140" text-anchor="middle" fill="black">
            %s
        </text>

        <!-- Localport -->
        <svg
            x="215"
            y="80"
            width="30"
            height="30"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
        >
            <rect x="16" y="16" width="6" height="6" rx="1" />
            <rect x="2" y="16" width="6" height="6" rx="1" />
            <rect x="9" y="2" width="6" height="6" rx="1" />
            <path d="M5 16v-3a1 1 0 0 1 1-1h12a1 1 0 0 1 1 1v3" />
            <path d="M12 12V8" />
        </svg>
        <text x="230" y="140" text-anchor="middle" fill="black">Localport</text>

        <!-- Internet -->
        <svg
            x="345"
            y="85"
            width="30"
            height="30"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
        >
            <circle cx="12" cy="12" r="10" />
            <path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20" />
            <path d="M2 12h20" />
        </svg>
        <text x="360" y="140" text-anchor="middle" fill="black">Internet</text>

        <!-- Connection - Local to Remote (Failing) -->
        <line
            x1="120"
            y1="100"
            x2="205"
            y2="100"
            stroke="black"
            stroke-width="2"
            stroke-dasharray="5,5"
        />

        <!-- Connection - Remote to Internet (OK) -->
        <line
            x1="250"
            y1="100"
            x2="335"
            y2="100"
            stroke="black"
            stroke-width="2"
        />
        </svg>
    </div>
    </body>
</html>
        `, endpoint))
}

func UnregisteredSubdomain(subdomain string) string {
	return Trim(fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
    <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Localport</title>
    <style>
        body {
        padding: 0 24px;
        margin: 0;
        height: 80vh;
        display: flex;
        justify-content: center;
        align-items: center;
        font-family: "Inter";
        }
        #main {
        display: flex;
        flex-direction: column;
        align-items: center;
        }
        h1,
        h2,
        h3 {
        font-weight: 300;
        }
        pre {
        font-size: 16px;
        font-family: monospace;
        padding: 8px;
        border-radius: 10px;
        border: 1px solid #ccc;
        }
    </style>
    </head>
    <body>
    <div id="main">
        <h1>The provided tunnel subdomain is not registered</h1>
        <h2>Start a new tunnel connection using the following command:</h2>
        <pre>localport http -p 8000 -s %s</pre>
        <h3>
        Checkout
        <a href="https://localport.app" target="_blank" rel="noreferrer"
            >localport.app</a
        >
        for more
        </h3>
    </div>
    </body>
</html>
        `, subdomain))
}
