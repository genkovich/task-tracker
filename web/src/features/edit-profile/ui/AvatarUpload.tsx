import { useRef, useState } from "react";
import { Trash2, Upload } from "lucide-react";
import { toast } from "sonner";
import { useAuth } from "@/app/providers/auth";
import { profileApi } from "@/features/edit-profile/api/profileApi";
import { ApiClientError } from "@/shared/api/client";
import { getDisplayName, getInitials } from "@/shared/lib/user";
import { Button } from "@/shared/ui/button";
import { UserAvatar } from "@/shared/ui/UserAvatar";

const MAX_BYTES = 2 * 1024 * 1024;
const ACCEPT = "image/png,image/jpeg,image/webp";

export function AvatarUpload() {
  const { user, fetchUser } = useAuth();
  const inputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);

  if (!user) return null;

  const handleFile = async (file: File) => {
    if (file.size > MAX_BYTES) {
      toast.error("Avatar must be 2MB or smaller");
      return;
    }
    setUploading(true);
    try {
      await profileApi.uploadAvatar(file);
      await fetchUser();
      toast.success("Avatar updated");
    } catch (err) {
      if (err instanceof ApiClientError) toast.error(err.message);
      else toast.error("Failed to upload avatar");
    } finally {
      setUploading(false);
    }
  };

  const handleRemove = async () => {
    setUploading(true);
    try {
      await profileApi.deleteAvatar();
      await fetchUser();
      toast.success("Avatar removed");
    } catch (err) {
      if (err instanceof ApiClientError) toast.error(err.message);
      else toast.error("Failed to remove avatar");
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="flex items-center gap-4">
      <UserAvatar
        src={user.avatar_url}
        seed={user.email}
        initials={getInitials(user)}
        alt={getDisplayName(user)}
        className="size-16"
        fallbackClassName="text-lg"
      />
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <input
          ref={inputRef}
          type="file"
          accept={ACCEPT}
          className="hidden"
          onChange={(e) => {
            const file = e.target.files?.[0];
            if (file) void handleFile(file);
            e.target.value = "";
          }}
        />
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={uploading}
          onClick={() => inputRef.current?.click()}
        >
          <Upload className="size-4" />
          {uploading ? "Working..." : "Upload"}
        </Button>
        {user.avatar_url && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={uploading}
            onClick={handleRemove}
            className="text-destructive"
          >
            <Trash2 className="size-4" />
            Remove
          </Button>
        )}
        <p className="text-xs text-muted-foreground">PNG, JPG, or WebP — up to 2MB</p>
      </div>
    </div>
  );
}
