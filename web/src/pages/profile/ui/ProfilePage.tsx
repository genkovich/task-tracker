import { useAuth } from "@/app/providers/auth";
import { AvatarUpload } from "@/features/edit-profile/ui/AvatarUpload";
import { ProfileForm } from "@/features/edit-profile/ui/ProfileForm";
import { getDisplayName, getInitials } from "@/shared/lib/user";
import { Card, CardContent } from "@/shared/ui/card";
import { Badge } from "@/shared/ui/badge";
import { UserAvatar } from "@/shared/ui/UserAvatar";

export default function ProfilePage() {
  const { user } = useAuth();

  if (!user) return null;

  const initials = getInitials(user);
  const displayName = getDisplayName(user);
  const positionLine = [user.position, user.department].filter(Boolean).join(" · ");

  return (
    <>
      <div className="animate-in-fade">
        <h1 className="text-3xl font-semibold tracking-tight">Profile</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          How you appear to other people in your stream
        </p>
      </div>

      <Card className="mt-6 animate-in-fade-up">
        <CardContent className="flex flex-col gap-4 sm:flex-row sm:items-center">
          <UserAvatar
            src={user.avatar_url}
            seed={user.email}
            initials={initials}
            alt={displayName}
            className="size-16 shrink-0"
            fallbackClassName="text-lg"
          />
          <div className="min-w-0 flex-1 space-y-1">
            <div className="flex flex-wrap items-center gap-2">
              <p className="font-medium">{displayName}</p>
              {user.role === "admin" && (
                <Badge variant="secondary" className="text-[10px]">
                  Admin
                </Badge>
              )}
            </div>
            {positionLine && <p className="text-sm text-muted-foreground">{positionLine}</p>}
            <p className="text-sm text-muted-foreground">{user.email}</p>
            {user.bio && (
              <p className="mt-2 max-w-xl text-sm text-muted-foreground whitespace-pre-wrap">
                {user.bio}
              </p>
            )}
          </div>
        </CardContent>
      </Card>

      <Card className="mt-6 animate-in-fade-up">
        <CardContent>
          <AvatarUpload />
        </CardContent>
      </Card>

      <div className="mt-6 animate-in-fade-up">
        <ProfileForm user={user} />
      </div>
    </>
  );
}
