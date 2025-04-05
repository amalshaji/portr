<script lang="ts">
  import ErrorText from "$lib/components/ErrorText.svelte";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import { Input } from "$lib/components/ui/input";
  import { Label } from "$lib/components/ui/label";
  import { Switch } from "$lib/components/ui/switch";
  import { Textarea } from "$lib/components/ui/textarea";
  import { instanceSettings } from "$lib/store";
  import { LoaderCircle , AlertTriangle } from "lucide-svelte";
  import { onDestroy, onMount } from "svelte";
  import { toast } from "svelte-sonner";
  import { getContext } from "svelte";
  import * as Alert from "$lib/components/ui/alert";

  let team = getContext("team") as string;
  let smtpEnabled: boolean;

  let addMemberEmailTemplate: string,
    addMemberEmailSubject: string,
    isUpdating = false;
  let smtpHost = "",
    smtpPort: number;
  let smtpUsername = "",
    smtpPassword = "";
  let fromAddress = "";

  let smtpHostError = "",
    smtpUsernameError = "",
    smtpPasswordError = "",
    fromAddressError = "",
    addMemberEmailSubjectError = "",
    addMemberEmailTemplateError = "";

  const validateForm = () => {
    let hasErrors = false;
    if (smtpEnabled) {
      if (!smtpHost) {
        smtpHostError = "SMTP host is required";
        hasErrors = true;
      }
      if (!smtpUsername) {
        smtpUsernameError = "SMTP username is required";
        hasErrors = true;
      }
      if (!smtpPassword) {
        smtpPasswordError = "SMTP password is required";
        hasErrors = true;
      }
      if (!fromAddress) {
        fromAddressError = "From address is required";
        hasErrors = true;
      }
      if (!addMemberEmailSubject) {
        addMemberEmailSubjectError = "Email subject is required";
        hasErrors = true;
      }
      if (!addMemberEmailTemplate) {
        addMemberEmailTemplateError = "Email template is required";
        hasErrors = true;
      }
    }
    return !hasErrors;
  };

  const resetErrors = () => {
    smtpHostError = "";
    smtpUsernameError = "";
    smtpPasswordError = "";
    fromAddressError = "";
    addMemberEmailSubjectError = "";
    addMemberEmailTemplateError = "";
  };

  const getSettings = async () => {
    const res = await fetch("/api/v1/instance-settings/");
    instanceSettings.set(await res.json());
  };

  let settingsUnSubscriber = instanceSettings.subscribe((settings) => {
    addMemberEmailTemplate = settings?.add_user_email_body || "";
    addMemberEmailSubject = settings?.add_user_email_subject || "";
    smtpEnabled = settings?.smtp_enabled || false;
    smtpHost = settings?.smtp_host || "";
    smtpPort = settings?.smtp_port || 587;
    smtpUsername = settings?.smtp_username || "";
    smtpPassword = settings?.smtp_password || "";
    fromAddress = settings?.from_address || "";
  });

  const updateEmailSettings = async () => {
    resetErrors();
    if (!validateForm()) return;
    isUpdating = true;
    try {
      const res = await fetch("/api/v1/instance-settings/", {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          "x-team-slug": team,
        },
        body: JSON.stringify({
          smtp_enabled: smtpEnabled,
          smtp_host: smtpHost,
          smtp_port: smtpPort,
          smtp_username: smtpUsername,
          smtp_password: smtpPassword,
          from_address: fromAddress,
          add_user_email_subject: addMemberEmailSubject,
          add_user_email_body: addMemberEmailTemplate,
        }),
      });
      if (res.ok) {
        instanceSettings.set(await res.json());
        toast.success("Email settings updated");
      } else {
        toast.error("Something went wrong");
      }
    } catch (err) {
      console.error(err);
    } finally {
      isUpdating = false;
    }
  };

  onMount(() => {
    getSettings();
  });

  onDestroy(() => {
    settingsUnSubscriber();
  });
</script>

<div class="p-6">
  <h1 class="text-2xl font-bold tracking-tight mb-6">Email Settings</h1>

  <Alert.Root class="mb-6 border-amber-200 bg-amber-50">
    <AlertTriangle class="h-4 w-4 text-amber-700" />
    <Alert.Title class="text-amber-700">Instance-wide Settings</Alert.Title>
    <Alert.Description class="text-amber-700">
      These settings apply to the entire application instance. Any changes made here will affect all teams and users.
    </Alert.Description>
  </Alert.Root>

  <Card.Root class="rounded-sm border shadow-sm w-full">
    <Card.Header class="space-y-3">
      <Card.Title>Email Configuration</Card.Title>
      <Card.Description>Configure SMTP settings and email templates for notifications</Card.Description>
    </Card.Header>
    <Card.Content class="space-y-6">
      <div class="flex items-center gap-4">
        <Switch bind:checked={smtpEnabled} id="smtp-toggle" />
        <Label for="smtp-toggle" class="font-medium">Enable SMTP</Label>
      </div>

      <div class="space-y-6">
        <div>
          <h3 class="text-sm font-medium mb-3">SMTP Server Settings</h3>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <Label for="smtp_host">SMTP Host</Label>
              <Input
                disabled={!smtpEnabled}
                bind:value={smtpHost}
                id="smtp_host"
                placeholder="smtp.example.com"
                required
                class={smtpHostError && "border-red-500"}
              />
              {#if smtpHostError}
                <p class="text-red-500 text-xs mt-1">{smtpHostError}</p>
              {/if}
            </div>
            <div>
              <Label for="smtp_port">SMTP Port</Label>
              <Input
                disabled={!smtpEnabled}
                bind:value={smtpPort}
                type="number"
                id="smtp_port"
                placeholder="587"
              />
            </div>
          </div>
        </div>

        <div>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <Label for="smtp_username">SMTP Username</Label>
              <Input
                disabled={!smtpEnabled}
                bind:value={smtpUsername}
                id="smtp_username"
                placeholder="username"
                class={smtpUsernameError && "border-red-500"}
              />
              {#if smtpUsernameError}
                <p class="text-red-500 text-xs mt-1">{smtpUsernameError}</p>
              {/if}
            </div>
            <div>
              <Label for="smtp_password">SMTP Password</Label>
              <Input
                disabled={!smtpEnabled}
                bind:value={smtpPassword}
                type="password"
                id="smtp_password"
                placeholder="••••••••"
                class={smtpPasswordError && "border-red-500"}
              />
              {#if smtpPasswordError}
                <p class="text-red-500 text-xs mt-1">{smtpPasswordError}</p>
              {/if}
            </div>
          </div>
        </div>

        <div>
          <Label for="from_address">From Address</Label>
          <Input
            disabled={!smtpEnabled}
            bind:value={fromAddress}
            id="from_address"
            placeholder="noreply@example.com"
            class={fromAddressError && "border-red-500"}
          />
          {#if fromAddressError}
            <p class="text-red-500 text-xs mt-1">{fromAddressError}</p>
          {/if}
        </div>
      </div>

      <div class="border-t pt-6">
        <h3 class="text-sm font-medium mb-3">Email Templates</h3>

        <div class="space-y-4">
          <div>
            <Label for="add_member_template_subject">Add member email subject</Label>
            <Input
              disabled={!smtpEnabled}
              bind:value={addMemberEmailSubject}
              id="add_member_template_subject"
              placeholder="You've been invited to join a team"
              class={addMemberEmailSubjectError && "border-red-500"}
            />
            {#if addMemberEmailSubjectError}
              <ErrorText error={addMemberEmailSubjectError} />
            {/if}
          </div>
          <div>
            <Label for="add_member_template_body">Add member email body</Label>
            <Textarea
              disabled={!smtpEnabled}
              rows={10}
              bind:value={addMemberEmailTemplate}
              id="add_member_template_body"
              placeholder="Hello, you have been invited to join our team."
              class={addMemberEmailTemplateError && "border-red-500"}
            />
            {#if addMemberEmailTemplateError}
              <ErrorText error={addMemberEmailTemplateError} />
            {/if}
            <p class="text-xs text-gray-500 mt-2">
              Available variables: <code>&#123;&#123; email &#125;&#125;</code>, <code>&#123;&#123; teamName &#125;&#125;</code>, <code>&#123;&#123; dashboardUrl &#125;&#125;</code>
            </p>
          </div>
        </div>
      </div>
    </Card.Content>
    <Card.Footer>
      <Button on:click={updateEmailSettings} disabled={isUpdating}>
        {#if isUpdating}
          <LoaderCircle class="mr-2 h-4 w-4 animate-spin" />
        {/if}
        Save changes
      </Button>
    </Card.Footer>
  </Card.Root>
</div>