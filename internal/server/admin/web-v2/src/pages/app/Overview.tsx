import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { Copy, Terminal } from "lucide-react";
import { copyCodeToClipboard } from "@/lib/utils";

export default function Overview() {
  const { team } = useParams<{ team: string }>();
  const [setupScript, setSetupScript] = useState("");

  const getSetupScript = async () => {
    if (!team) return;
    try {
      const res = await fetch("/api/v1/config/setup-script", {
        headers: {
          "x-team-slug": team,
        },
      });
      const data = await res.json();
      setSetupScript(data.message || "");
    } catch (error) {
      console.error("Failed to fetch setup script:", error);
    }
  };

  const installCommand = `curl -sSf https://install.portr.dev | sh`;
  const homebrewCommand = `brew install amalshaji/taps/portr`;
  const helpCommand = "portr -h";

  const handleCopy = (text: string) => {
    copyCodeToClipboard(text);
  };

  useEffect(() => {
    getSetupScript();
  }, [team]);

  return (
    <div className="space-y-8">
      {/* Dashboard Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-black">
            Dashboard
          </h1>
          <p className="text-gray-600">Welcome to your {team} dashboard.</p>
        </div>
        <Terminal className="h-8 w-8 text-muted-foreground" />
      </div>

      {/* Client Setup Section */}
      <div className="rounded-lg border bg-card p-6">
        <div className="mb-6">
          <h2 className="text-xl font-semibold">Client Setup</h2>
          <p className="text-muted-foreground mt-1">
            Follow these steps to set up and configure the portr client
          </p>
        </div>
        <div className="space-y-6">
          <div className="rounded-lg border bg-muted/50 p-6">
            <h3 className="text-sm font-medium mb-3 flex items-center gap-2">
              <span className="flex h-6 w-6 rounded-full bg-primary text-primary-foreground items-center justify-center text-xs font-semibold">
                1
              </span>
              Install the portr client
            </h3>

            <div className="space-y-4">
              <div>
                <p className="text-sm text-muted-foreground mb-2">
                  Using the install script:
                </p>
                <div className="relative group">
                  <pre className="bg-muted p-3 rounded-lg text-sm font-mono overflow-x-auto">
                    {installCommand}
                  </pre>
                  <button
                    className="absolute right-2 top-2 p-1.5 bg-background border rounded opacity-0 group-hover:opacity-100 transition-opacity hover:bg-muted"
                    onClick={() => handleCopy(installCommand)}
                  >
                    <Copy className="h-3 w-3" />
                  </button>
                </div>
              </div>

              <div>
                <p className="text-sm text-muted-foreground mb-2">
                  Or using homebrew:
                </p>
                <div className="relative group">
                  <pre className="bg-muted p-3 rounded-lg text-sm font-mono overflow-x-auto">
                    {homebrewCommand}
                  </pre>
                  <button
                    className="absolute right-2 top-2 p-1.5 bg-background border rounded opacity-0 group-hover:opacity-100 transition-opacity hover:bg-muted"
                    onClick={() => handleCopy(homebrewCommand)}
                  >
                    <Copy className="h-3 w-3" />
                  </button>
                </div>
              </div>
            </div>

            <p className="mt-4 text-sm text-muted-foreground">
              You can also download the binary from the{" "}
              <a
                href="https://github.com/amalshaji/portr/releases"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline font-medium"
              >
                GitHub releases
              </a>
            </p>
          </div>

          <div className="rounded-lg border bg-muted/50 p-6">
            <h3 className="text-sm font-medium mb-3 flex items-center gap-2">
              <span className="flex h-6 w-6 rounded-full bg-primary text-primary-foreground items-center justify-center text-xs font-semibold">
                2
              </span>
              Run the following command to set up portr client auth
            </h3>

            <div className="relative group">
              <pre className="bg-muted p-3 rounded-lg text-sm font-mono overflow-x-auto">
                {setupScript}
              </pre>
              <button
                className="absolute right-2 top-2 p-1.5 bg-background border rounded opacity-0 group-hover:opacity-100 transition-opacity hover:bg-muted"
                onClick={() => handleCopy(setupScript)}
              >
                <Copy className="h-3 w-3" />
              </button>
            </div>

            <p className="mt-4 text-sm text-muted-foreground">
              Note: use{" "}
              <code className="bg-muted px-1 py-0.5 rounded text-sm">
                ./portr
              </code>{" "}
              instead of{" "}
              <code className="bg-muted px-1 py-0.5 rounded text-sm">
                portr
              </code>{" "}
              if the binary is in the same folder and not set in{" "}
              <code className="bg-muted px-1 py-0.5 rounded text-sm">
                $PATH
              </code>
            </p>
          </div>

          <div className="rounded-lg border bg-muted/50 p-6">
            <h3 className="text-sm font-medium mb-3 flex items-center gap-2">
              <span className="flex h-6 w-6 rounded-full bg-primary text-primary-foreground items-center justify-center text-xs font-semibold">
                3
              </span>
              You're ready to use the tunnel
            </h3>

            <p className="text-muted-foreground text-sm">
              Run{" "}
              <code className="bg-muted px-1 py-0.5 rounded text-sm">
                {helpCommand}
              </code>{" "}
              or check out the{" "}
              <a
                href="https://portr.dev/docs/client"
                target="_blank"
                rel="noopener noreferrer"
                className="text-primary hover:underline font-medium"
              >
                client documentation
              </a>{" "}
              for more information.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
