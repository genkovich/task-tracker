import { useState } from "react";
import type { CurrentUser } from "@/entities/user/model/types";
import { profileApi } from "@/features/edit-profile/api/profileApi";
import { useAuth } from "@/app/providers/auth";
import { ApiClientError } from "@/shared/api/client";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import { Textarea } from "@/shared/ui/textarea";
import { Button } from "@/shared/ui/button";
import { TimezoneCombobox } from "./TimezoneCombobox";

interface ProfileFormProps {
  user: CurrentUser;
}

export function ProfileForm({ user }: ProfileFormProps) {
  const { fetchUser } = useAuth();

  const [form, setForm] = useState({
    first_name: user.first_name ?? "",
    last_name: user.last_name ?? "",
    position: user.position ?? "",
    department: user.department ?? "",
    bio: user.bio ?? "",
    timezone: user.timezone ?? "",
  });
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function update(field: keyof typeof form, value: string) {
    setForm((prev) => ({ ...prev, [field]: value }));
    setSaved(false);
    setError(null);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError(null);
    setSaved(false);

    try {
      await profileApi.updateProfile({
        first_name: form.first_name || null,
        last_name: form.last_name || null,
        position: form.position || null,
        department: form.department || null,
        bio: form.bio || null,
        timezone: form.timezone || null,
      });
      await fetchUser();
      setSaved(true);
    } catch (err) {
      if (err instanceof ApiClientError) {
        setError(err.message);
      } else {
        setError("Something went wrong");
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Edit profile</CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="first_name">First name</Label>
            <Input
              id="first_name"
              value={form.first_name}
              onChange={(e) => update("first_name", e.target.value)}
              placeholder="Jane"
              maxLength={100}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="last_name">Last name</Label>
            <Input
              id="last_name"
              value={form.last_name}
              onChange={(e) => update("last_name", e.target.value)}
              placeholder="Doe"
              maxLength={100}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="position">Position</Label>
            <Input
              id="position"
              value={form.position}
              onChange={(e) => update("position", e.target.value)}
              placeholder="Senior Engineer"
              maxLength={200}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="department">Department</Label>
            <Input
              id="department"
              value={form.department}
              onChange={(e) => update("department", e.target.value)}
              placeholder="Platform"
              maxLength={200}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="timezone">Timezone</Label>
            <TimezoneCombobox
              id="timezone"
              value={form.timezone}
              onChange={(tz) => update("timezone", tz)}
            />
          </div>

          <div className="space-y-2 sm:col-span-2">
            <Label htmlFor="bio">Bio</Label>
            <Textarea
              id="bio"
              value={form.bio}
              onChange={(e) => update("bio", e.target.value)}
              placeholder="A few words about yourself"
              rows={3}
            />
          </div>

          <div className="flex items-center gap-3 sm:col-span-2">
            <Button type="submit" disabled={saving}>
              {saving ? "Saving..." : "Save"}
            </Button>
            {saved && (
              <span className="text-sm text-muted-foreground">Saved</span>
            )}
            {error && (
              <span className="text-sm text-destructive">{error}</span>
            )}
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
