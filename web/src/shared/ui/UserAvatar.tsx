import { getAvatarGradient, getAvatarTextColor } from "@/shared/lib/avatarGradient";
import { cn } from "@/shared/lib/utils";
import { Avatar, AvatarFallback, AvatarImage } from "@/shared/ui/avatar";

interface UserAvatarProps {
  readonly src?: string | null;
  readonly seed: string;
  readonly initials: string;
  readonly alt?: string;
  readonly className?: string;
  readonly fallbackClassName?: string;
}

export function UserAvatar({
  src,
  seed,
  initials,
  alt,
  className,
  fallbackClassName,
}: UserAvatarProps) {
  return (
    <Avatar className={className}>
      {src && <AvatarImage src={src} alt={alt ?? initials} />}
      <AvatarFallback
        className={cn("font-medium text-white", fallbackClassName)}
        style={{
          background: getAvatarGradient(seed),
          color: getAvatarTextColor(),
        }}
      >
        {initials}
      </AvatarFallback>
    </Avatar>
  );
}
