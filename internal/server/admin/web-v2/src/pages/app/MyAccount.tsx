import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useUserStore } from "@/lib/store";
import {
  Copy,
  Eye,
  EyeOff,
  KeyRound,
  KeySquare,
  LoaderCircle,
  User,
} from "lucide-react";
import { toast } from "sonner";
import { copyCodeToClipboard } from "@/lib/utils";

export default function MyAccount() {
  const { team } = useParams<{ team: string }>();
  const { currentUser, setCurrentUser } = useUserStore();

  const refetchCurrentUser = async () => {
    if (!team) return;
    try {
      const response = await fetch("/api/v1/user/me", {
        headers: {
          "Content-Type": "application/json",
          "x-team-slug": team,
        },
      });
      if (response.ok) {
        const userData = await response.json();
        setCurrentUser(userData);
      }
    } catch (err) {
      console.error("Failed to refetch user:", err);
    }
  };

  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [passwordError, setPasswordError] = useState("");

  const [isUpdating, setIsUpdating] = useState(false);
  const [isChangingPassword, setIsChangingPassword] = useState(false);
  const [isRotatingSecretKey, setIsRotatingSecretKey] = useState(false);
  const [secretKeyVisible, setSecretKeyVisible] = useState(false);

  useEffect(() => {
    if (currentUser) {
      setFirstName(currentUser?.user?.first_name || "");
      setLastName(currentUser?.user?.last_name || "");
    }
  }, [currentUser]);

  const updateProfile = async () => {
    setIsUpdating(true);
    try {
      const res = await fetch("/api/v1/user/me/update", {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          "x-team-slug": team || "",
        },
        body: JSON.stringify({
          first_name: firstName,
          last_name: lastName,
        }),
      });

      if (res.ok) {
        toast.success("Profile updated");
        await refetchCurrentUser(); // Refetch user data after profile update
      } else {
        toast.error("Something went wrong");
      }
    } catch (err) {
      console.error(err);
      toast.error("Something went wrong");
    } finally {
      setIsUpdating(false);
    }
  };

  const changePassword = async () => {
    setPasswordError("");

    if (password !== confirmPassword) {
      setPasswordError("Passwords do not match");
      return;
    }

    setIsChangingPassword(true);

    try {
      const res = await fetch("/api/v1/user/me/change-password", {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          "x-team-slug": team || "",
        },
        body: JSON.stringify({ password }),
      });

      if (res.ok) {
        toast.success("Password changed");
        setPassword("");
        setConfirmPassword("");
        // No need to refetch for password change as it doesn't affect displayed data
      } else {
        toast.error("Something went wrong");
      }
    } catch (err) {
      console.error(err);
      toast.error("Something went wrong");
    } finally {
      setIsChangingPassword(false);
    }
  };

  const rotateSecretKey = async () => {
    setIsRotatingSecretKey(true);
    try {
      const res = await fetch("/api/v1/user/me/rotate-secret-key", {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          "x-team-slug": team || "",
        },
      });

      if (res.ok) {
        toast.success("New secret key generated");
        await refetchCurrentUser(); // Refetch user data after key rotation
      } else {
        toast.error("Something went wrong");
      }
    } catch (err) {
      console.error(err);
      toast.error("Something went wrong");
    } finally {
      setIsRotatingSecretKey(false);
    }
  };

  const copySecretKey = () => {
    if (currentUser?.secret_key) {
      copyCodeToClipboard(String(currentUser.secret_key));
      toast.success("Secret key copied to clipboard");
    }
  };

  const secretKey = currentUser?.secret_key ? String(currentUser.secret_key) : "";

  return (
    <div className="mx-auto w-full max-w-3xl space-y-6">
      <Section
        Icon={User}
        title="Profile"
        description="Your name as it appears to the rest of the team."
      >
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="first_name">First name</Label>
            <Input
              type="text"
              id="first_name"
              autoComplete="given-name"
              placeholder="Ada"
              value={firstName}
              onChange={(e) => setFirstName(e.target.value)}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="last_name">Last name</Label>
            <Input
              type="text"
              id="last_name"
              autoComplete="family-name"
              placeholder="Lovelace"
              value={lastName}
              onChange={(e) => setLastName(e.target.value)}
            />
          </div>
        </div>

        <Button onClick={updateProfile} disabled={isUpdating} size="sm">
          {isUpdating && <LoaderCircle className="size-4 animate-spin" />}
          Save profile
        </Button>
      </Section>

      <Section
        Icon={KeySquare}
        title="Password"
        description="Used to sign in to this console."
      >
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="password">New password</Label>
            <Input
              type="password"
              id="password"
              autoComplete="new-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="confirm_password">Confirm password</Label>
            <Input
              type="password"
              id="confirm_password"
              autoComplete="new-password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              aria-invalid={passwordError ? true : undefined}
              aria-describedby={passwordError ? "confirm-password-error" : undefined}
            />
            {passwordError && (
              <p
                id="confirm-password-error"
                role="alert"
                className="text-xs text-destructive"
              >
                {passwordError}
              </p>
            )}
          </div>
        </div>

        <Button
          onClick={changePassword}
          disabled={isChangingPassword || !password}
          size="sm"
        >
          {isChangingPassword && (
            <LoaderCircle className="size-4 animate-spin" />
          )}
          Update password
        </Button>
      </Section>

      <Section
        Icon={KeyRound}
        title="Client secret key"
        description={
          <>
            Authenticates the Portr client against{" "}
            <span className="data">{team}</span>.
          </>
        }
      >
        <div className="flex gap-2">
          <div className="relative flex-1">
            {/* Masked by default: this is a live credential and the console is
                often open on a shared screen. */}
            <Input
              type={secretKeyVisible ? "text" : "password"}
              readOnly
              aria-label="Client secret key"
              value={secretKey}
              className="data pr-10 text-sm"
            />
            <Button
              variant="ghost"
              size="sm"
              aria-label={secretKeyVisible ? "Hide secret key" : "Show secret key"}
              aria-pressed={secretKeyVisible}
              className="absolute top-1/2 right-1 size-7 -translate-y-1/2 p-0"
              onClick={() => setSecretKeyVisible((visible) => !visible)}
            >
              {secretKeyVisible ? (
                <EyeOff className="size-4" />
              ) : (
                <Eye className="size-4" />
              )}
            </Button>
          </div>
          <Button
            variant="outline"
            size="sm"
            className="h-9"
            onClick={copySecretKey}
          >
            <Copy className="size-4" />
            Copy
          </Button>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="outline" size="sm" disabled={isRotatingSecretKey}>
                {isRotatingSecretKey && (
                  <LoaderCircle className="size-4 animate-spin" />
                )}
                Rotate key
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Rotate your secret key?</AlertDialogTitle>
                <AlertDialogDescription>
                  The current key stops working immediately. Every machine using
                  it has to be re-authenticated with the new key before it can
                  open a tunnel.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction onClick={rotateSecretKey}>
                  Rotate key
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
          <p className="text-xs text-muted-foreground">
            Rotating invalidates the previous key immediately.
          </p>
        </div>
      </Section>
    </div>
  );
}

function Section({
  Icon,
  title,
  description,
  children,
}: {
  Icon: typeof User;
  title: string;
  description: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section className="overflow-hidden rounded-md border border-border bg-card">
      <header className="flex items-start gap-3 border-b border-border px-4 py-3">
        <Icon className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
        <div>
          <h2 className="text-sm font-semibold">{title}</h2>
          <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>
        </div>
      </header>
      <div className="space-y-4 p-4">{children}</div>
    </section>
  );
}
