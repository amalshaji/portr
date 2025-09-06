import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { LoaderCircle, Copy, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "sonner";
import { useUserStore } from "@/lib/store";

interface InviteUserDialogProps {
  isOpen: boolean;
  setIsOpen: (open: boolean) => void;
  onSuccess?: () => void;
}

export default function InviteUserDialog({
  isOpen,
  setIsOpen,
  onSuccess,
}: InviteUserDialogProps) {
  const { team } = useParams<{ team: string }>();
  const { currentUser } = useUserStore();

  const [email, setEmail] = useState("");
  const [role, setRole] = useState("member");
  const [setSuperuser, setSetSuperuser] = useState(false);

  // Update role to admin when superuser is selected
  useEffect(() => {
    if (setSuperuser) {
      setRole("admin");
    }
  }, [setSuperuser]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");
  const [generatedPassword, setGeneratedPassword] = useState("");
  const [showPasswordDialog, setShowPasswordDialog] = useState(false);
  const [copied, setCopied] = useState(false);

  const roles = [
    { value: "member", label: "Member" },
    { value: "admin", label: "Admin" },
  ];

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (!email.trim()) {
      setError("Email is required");
      return;
    }

    setIsLoading(true);
    try {
      const res = await fetch("/api/v1/team/add", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "x-team-slug": team || "",
        },
        body: JSON.stringify({
          email: email.trim(),
          role,
          set_superuser: setSuperuser,
        }),
      });

      if (res.ok) {
        const { team_user, password } = await res.json();

        if (team_user) {
          toast.success(`${email} added to team`);
          onSuccess?.(); // Refetch data after successful user addition
        }

        // Reset form
        setEmail("");
        setRole("member");
        setSetSuperuser(false);
        setIsOpen(false);

        if (password) {
          setGeneratedPassword(password);
          setShowPasswordDialog(true);
        }

        // TODO: Refresh users list in parent component
        // This would need to be handled by the parent component
      } else {
        const errorData = await res.json();
        setError(errorData.message || "Failed to add user");
      }
    } catch (err) {
      console.error("Error adding user:", err);
      setError("Failed to add user");
    } finally {
      setIsLoading(false);
    }
  };

  const handleCopyPassword = async () => {
    try {
      await navigator.clipboard.writeText(generatedPassword);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy password:", err);
    }
  };

  const handleClosePasswordDialog = () => {
    setShowPasswordDialog(false);
    setGeneratedPassword("");
    setCopied(false);
  };

  return (
    <>
      <Dialog open={isOpen} onOpenChange={setIsOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add member</DialogTitle>
            <DialogDescription>
              Invite a new member to join your team
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
              <div className="p-3 text-sm text-red-600 bg-red-50 border border-red-200 rounded-md">
                {error}
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                placeholder="user@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="role">Role</Label>
              <Select
                value={role}
                onValueChange={setRole}
                disabled={setSuperuser}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select a role" />
                </SelectTrigger>
                <SelectContent>
                  {roles.map((roleOption) => (
                    <SelectItem key={roleOption.value} value={roleOption.value}>
                      {roleOption.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {currentUser?.user?.is_superuser && (
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="set_superuser"
                  checked={setSuperuser}
                  onCheckedChange={(checked) =>
                    setSetSuperuser(checked as boolean)
                  }
                />
                <Label
                  htmlFor="set_superuser"
                  className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
                >
                  Make superuser
                </Label>
              </div>
            )}

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setIsOpen(false)}
                disabled={isLoading}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={isLoading}>
                {isLoading && (
                  <LoaderCircle className="mr-2 h-4 w-4 animate-spin" />
                )}
                Add Member
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Password Dialog */}
      <Dialog
        open={showPasswordDialog}
        onOpenChange={handleClosePasswordDialog}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Generated Password</DialogTitle>
            <DialogDescription>
              Here's the generated password for the new user. Make sure to save
              it securely.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="flex items-center space-x-2">
              <Input value={generatedPassword} readOnly className="font-mono" />
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleCopyPassword}
                className="shrink-0"
              >
                {copied ? (
                  <Check className="h-4 w-4" />
                ) : (
                  <Copy className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>

          <DialogFooter>
            <Button onClick={handleClosePasswordDialog}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
