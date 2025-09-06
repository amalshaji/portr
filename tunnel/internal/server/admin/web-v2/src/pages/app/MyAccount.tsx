import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useUserStore } from "@/lib/store";
import { LoaderCircle, User, KeySquare, KeyRound, Copy } from "lucide-react";
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

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold tracking-tight">
          Account & Settings
        </h1>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-xl">Profile Information</CardTitle>
          <CardDescription>Update your personal details</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="bg-muted/50 rounded-lg p-6 border space-y-4">
            <div className="flex items-center gap-3">
              <User className="h-5 w-5 text-primary" />
              <div>
                <h3 className="text-sm font-medium">Personal Details</h3>
                <p className="text-xs text-muted-foreground">
                  Your name as it appears across the platform
                </p>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-x-6 gap-y-6 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="first_name">First Name</Label>
                <Input
                  type="text"
                  id="first_name"
                  placeholder="John"
                  value={firstName}
                  onChange={(e) => setFirstName(e.target.value)}
                  className="bg-background"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="last_name">Last Name</Label>
                <Input
                  type="text"
                  id="last_name"
                  placeholder="Doe"
                  value={lastName}
                  onChange={(e) => setLastName(e.target.value)}
                  className="bg-background"
                />
              </div>
            </div>

            <Button
              onClick={updateProfile}
              disabled={isUpdating}
              className="mt-2"
            >
              {isUpdating && (
                <LoaderCircle className="mr-2 h-4 w-4 animate-spin" />
              )}
              Save Profile
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-xl">Security</CardTitle>
          <CardDescription>
            Manage your security settings and credentials
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="bg-muted/50 rounded-lg p-6 border space-y-4">
            <div className="flex items-center gap-3">
              <KeySquare className="h-5 w-5 text-primary" />
              <div>
                <h3 className="text-sm font-medium">Change Password</h3>
                <p className="text-xs text-muted-foreground">
                  Update your login credentials
                </p>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-x-6 gap-y-6 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="password">New Password</Label>
                <Input
                  type="password"
                  id="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="bg-background"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="confirm_password">Confirm Password</Label>
                <Input
                  type="password"
                  id="confirm_password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className={`bg-background ${
                    passwordError ? "border-destructive" : ""
                  }`}
                />
                {passwordError && (
                  <p className="text-destructive text-xs mt-1">
                    {passwordError}
                  </p>
                )}
              </div>
            </div>

            <Button
              onClick={changePassword}
              disabled={isChangingPassword || !password}
              className="mt-2"
            >
              {isChangingPassword && (
                <LoaderCircle className="mr-2 h-4 w-4 animate-spin" />
              )}
              Update Password
            </Button>
          </div>

          <div className="bg-muted/50 rounded-lg p-6 border space-y-4">
            <div className="flex items-center gap-3">
              <KeyRound className="h-5 w-5 text-primary" />
              <div>
                <h3 className="text-sm font-medium">API Secret Key</h3>
                <p className="text-xs text-muted-foreground">
                  Used to authenticate client connections for team:{" "}
                  <span className="font-medium">{team}</span>
                </p>
              </div>
            </div>

            <div className="relative">
              <Input
                type="text"
                readOnly
                value={currentUser?.secret_key || ""}
                className="pr-10 font-mono text-sm bg-background"
              />
              <Button
                variant="ghost"
                size="sm"
                className="absolute right-1 top-1/2 -translate-y-1/2 h-8 w-8 p-0"
                onClick={copySecretKey}
              >
                <Copy className="h-4 w-4" />
              </Button>
            </div>

            <div>
              <Button
                variant="outline"
                onClick={rotateSecretKey}
                disabled={isRotatingSecretKey}
              >
                {isRotatingSecretKey && (
                  <LoaderCircle className="mr-2 h-4 w-4 animate-spin" />
                )}
                Rotate Key
              </Button>
              <p className="text-xs text-muted-foreground mt-2">
                Rotating your key will invalidate your previous key immediately
              </p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
