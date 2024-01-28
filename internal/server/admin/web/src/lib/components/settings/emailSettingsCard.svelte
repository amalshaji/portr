<script lang="ts">
  import { Label } from "$lib/components/ui/label";
  import { Button } from "$lib/components/ui/button";
  import * as Card from "$lib/components/ui/card";
  import { Textarea } from "$lib/components/ui/textarea";
  import { settings } from "$lib/store";
  import { onDestroy } from "svelte";
  import { toast } from "svelte-sonner";
  import { Reload } from "radix-icons-svelte";
  import { Switch } from "$lib/components/ui/switch";
  import { Input } from "$lib/components/ui/input";
  import ErrorText from "../ErrorText.svelte";

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

  let settingsUnSubscriber = settings.subscribe((settings) => {
    addMemberEmailTemplate = settings?.AddMemberEmailTemplate || "";
    addMemberEmailSubject = settings?.AddMemberEmailSubject || "";
    smtpEnabled = settings?.SmtpEnabled || false;
    smtpHost = settings?.SmtpHost || "";
    smtpPort = settings?.SmtpPort || 587;
    smtpUsername = settings?.SmtpUsername || "";
    smtpPassword = settings?.SmtpPassword || "";
    fromAddress = settings?.FromAddress || "";
  });

  const updateEmailSettings = async () => {
    resetErrors();
    if (!validateForm()) return;
    isUpdating = true;
    try {
      const res = await fetch("/api/setting/email/update", {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          SmtpEnabled: smtpEnabled,
          SmtpHost: smtpHost,
          SmtpPort: smtpPort,
          SmtpUsername: smtpUsername,
          SmtpPassword: smtpPassword,
          FromAddress: fromAddress,
          addMemberEmailSubject: addMemberEmailSubject,
          addMemberEmailTemplate: addMemberEmailTemplate,
        }),
      });
      if (res.ok) {
        settings.set(await res.json());
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

  onDestroy(() => {
    settingsUnSubscriber();
  });
</script>

<Card.Root class="rounded-sm border-none shadow-none">
  <Card.Header class="space-y-3">
    <Card.Title>Email Settings</Card.Title>
    <Card.Description>Configure email settings</Card.Description>
  </Card.Header>
  <Card.Content class="space-y-2">
    <div class="space-y-4">
      <div class="flex items-center gap-4">
        <Label for="invite_email_template">Enable SMTP</Label>
        <Switch bind:checked={smtpEnabled} />
      </div>
      <div class="grid grid-cols-4 gap-2">
        <div>
          <Label for="smtp_host">SMTP Host</Label>
          <Input
            disabled={!smtpEnabled}
            bind:value={smtpHost}
            id="smtp_host"
            placeholder="smtp.portr.dev"
            required
            class={smtpHostError && "border-red-500"}
          />
          {#if smtpHostError}
            <p class="text-red-500 text-xs italic">{smtpHostError}</p>
          {/if}
        </div>
        <div>
          <Label for="smtp_host">SMTP Port</Label>
          <Input
            disabled={!smtpEnabled}
            bind:value={smtpPort}
            type="number"
            id="smtp_host"
            placeholder="587"
          />
        </div>
      </div>
      <div class="grid grid-cols-4 gap-2">
        <div>
          <Label for="smtp_username">SMTP Username</Label>
          <Input
            disabled={!smtpEnabled}
            bind:value={smtpUsername}
            id="smtp_username"
            placeholder="portr"
            class={smtpUsernameError && "border-red-500"}
          />
          {#if smtpUsernameError}
            <p class="text-red-500 text-xs italic">{smtpUsernameError}</p>
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
            <p class="text-red-500 text-xs italic">{smtpPasswordError}</p>
          {/if}
        </div>
      </div>
      <div class="w-1/2">
        <Label for="from_address">From Address</Label>
        <Input
          disabled={!smtpEnabled}
          bind:value={fromAddress}
          id="from_address"
          placeholder="hey@portr.dev"
          class={fromAddressError && "border-red-500"}
        />
        {#if fromAddressError}
          <p class="text-red-500 text-xs italic">{fromAddressError}</p>
        {/if}
      </div>
      <div>
        <Label for="add_member_template_subject">Add member email subject</Label
        >
        <Input
          disabled={!smtpEnabled}
          bind:value={addMemberEmailSubject}
          id="add_member_template_subject"
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
          class={addMemberEmailTemplateError && "border-red-500"}
        />
        {#if addMemberEmailTemplateError}
          <ErrorText error={addMemberEmailTemplateError} />
        {/if}
      </div>
    </div>
  </Card.Content>
  <Card.Footer>
    <Button on:click={updateEmailSettings} disabled={isUpdating}>
      {#if isUpdating}
        <Reload class="mr-2 h-4 w-4 animate-spin" />
      {/if}
      Save changes
    </Button>
  </Card.Footer>
</Card.Root>
