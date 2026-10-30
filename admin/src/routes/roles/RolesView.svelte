<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/client';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';

  interface RolePermission {
    resource: string;
    action: string;
  }

  interface CustomRole {
    id: string;
    name: string;
    description: string;
    permissions: RolePermission[];
    created_at?: string;
  }

  let roles = $state<CustomRole[]>([]);
  let loading = $state(true);
  let statusMsg = $state<string | null>(null);
  let statusErr = $state<string | null>(null);

  // Edit / Create Modal state
  let modalOpen = $state(false);
  let isEditing = $state(false);
  let formId = $state('');
  let formName = $state('');
  let formDesc = $state('');
  let formPermissions = $state<RolePermission[]>([]);
  let saving = $state(false);

  // New permission row inputs
  let newResource = $state('collection:*');
  let newAction = $state('read');

  // Delete modal state
  let deleteTarget = $state<CustomRole | null>(null);
  let deleting = $state(false);

  async function loadRoles() {
    loading = true;
    try {
      const res = await api.get<CustomRole[]>('/api/v1/admin/roles');
      if (res) {
        roles = Array.isArray(res) ? res : (((res as any).data as CustomRole[]) || []);
      }
    } catch (err: any) {
      statusErr = err?.message || 'Failed to load roles';
    } finally {
      loading = false;
    }
  }

  function openCreateModal() {
    isEditing = false;
    formId = '';
    formName = '';
    formDesc = '';
    formPermissions = [
      { resource: 'collection:*', action: 'read' },
    ];
    modalOpen = true;
  }

  function openEditModal(role: CustomRole) {
    isEditing = true;
    formId = role.id;
    formName = role.name;
    formDesc = role.description || '';
    formPermissions = [...(role.permissions || [])];
    modalOpen = true;
  }

  function addPermission() {
    if (!newResource.trim() || !newAction.trim()) return;
    const exists = formPermissions.some(
      (p) => p.resource === newResource.trim() && p.action === newAction.trim()
    );
    if (!exists) {
      formPermissions = [...formPermissions, { resource: newResource.trim(), action: newAction.trim() }];
    }
  }

  function removePermission(index: number) {
    formPermissions = formPermissions.filter((_, i) => i !== index);
  }

  async function handleSaveRole() {
    if (!formId.trim() || !formName.trim()) {
      statusErr = 'Role ID and Name are required';
      return;
    }
    saving = true;
    statusErr = null;
    statusMsg = null;
    try {
      const payload = {
        id: formId.trim().toLowerCase(),
        name: formName.trim(),
        description: formDesc.trim(),
        permissions: formPermissions,
      };
      await api.post('/api/v1/admin/roles', payload);
      statusMsg = `Role ${formName} saved successfully`;
      modalOpen = false;
      await loadRoles();
    } catch (err: any) {
      statusErr = err?.message || 'Failed to save role';
    } finally {
      saving = false;
    }
  }

  async function handleDeleteRole() {
    if (!deleteTarget) return;
    deleting = true;
    statusErr = null;
    statusMsg = null;
    try {
      await api.delete(`/api/v1/admin/roles/${deleteTarget.id}`);
      statusMsg = `Role ${deleteTarget.name} deleted`;
      deleteTarget = null;
      await loadRoles();
    } catch (err: any) {
      statusErr = err?.message || 'Failed to delete role';
    } finally {
      deleting = false;
    }
  }

  onMount(() => {
    loadRoles();
  });
</script>

<div class="space-y-6">
  <!-- Header -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <div>
      <h2 class="text-2xl font-bold tracking-tight text-foreground">Custom Roles & Permissions</h2>
      <p class="text-sm text-muted-foreground">Manage granular tenant roles and resource capabilities.</p>
    </div>
    <Button onclick={openCreateModal}>
      <svg class="size-4 mr-2" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M12 5v14M5 12h14" />
      </svg>
      Create Role
    </Button>
  </div>

  <!-- Alerts -->
  {#if statusMsg}
    <div class="rounded-lg border border-green-500/30 bg-green-500/10 p-4 text-sm text-green-700 dark:text-green-400">
      {statusMsg}
    </div>
  {/if}
  {#if statusErr}
    <div class="rounded-lg border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
      {statusErr}
    </div>
  {/if}

  <!-- Roles Table -->
  <div class="rounded-xl border border-border bg-card shadow-xs overflow-hidden">
    {#if loading}
      <div class="p-8 text-center text-sm text-muted-foreground">Loading custom roles...</div>
    {:else if roles.length === 0}
      <div class="p-12 text-center space-y-3">
        <div class="inline-flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
          <svg class="size-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect width="18" height="18" x="3" y="3" rx="2" />
            <path d="M7 7h10" /><path d="M7 12h10" /><path d="M7 17h10" />
          </svg>
        </div>
        <h3 class="text-base font-semibold text-foreground">No Custom Roles Defined</h3>
        <p class="text-sm text-muted-foreground max-w-sm mx-auto">
          Create granular roles to give team members scoped access to specific content types and operations.
        </p>
        <Button variant="outline" size="sm" onclick={openCreateModal}>Create First Role</Button>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-sm">
          <thead class="border-b border-border bg-muted/40 text-xs uppercase text-muted-foreground">
            <tr>
              <th class="px-4 py-3 font-medium">Role</th>
              <th class="px-4 py-3 font-medium">Identifier</th>
              <th class="px-4 py-3 font-medium">Permissions</th>
              <th class="px-4 py-3 text-right font-medium">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            {#each roles as role}
              <tr class="hover:bg-muted/30 transition-colors">
                <td class="px-4 py-3 font-medium text-foreground">
                  <div>{role.name}</div>
                  {#if role.description}
                    <div class="text-xs text-muted-foreground">{role.description}</div>
                  {/if}
                </td>
                <td class="px-4 py-3 text-muted-foreground font-mono text-xs">
                  {role.id}
                </td>
                <td class="px-4 py-3">
                  <div class="flex flex-wrap gap-1.5 max-w-md">
                    {#if role.permissions && role.permissions.length > 0}
                      {#each role.permissions as perm}
                        <Badge variant="secondary" class="text-xs">
                          {perm.resource}:{perm.action}
                        </Badge>
                      {/each}
                    {:else}
                      <span class="text-xs text-muted-foreground italic">No permissions</span>
                    {/if}
                  </div>
                </td>
                <td class="px-4 py-3 text-right space-x-2">
                  <Button variant="ghost" size="sm" onclick={() => openEditModal(role)}>Edit</Button>
                  <Button variant="ghost" size="sm" class="text-destructive hover:text-destructive" onclick={() => (deleteTarget = role)}>Delete</Button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>

<!-- Create / Edit Role Modal -->
<Modal open={modalOpen} onclose={() => (modalOpen = false)} title={isEditing ? 'Edit Role' : 'Create Custom Role'}>
  <div class="space-y-4">
    <div>
      <label for="role-id-input" class="block text-xs font-medium text-foreground mb-1">Role Identifier</label>
      <Input
        id="role-id-input"
        bind:value={formId}
        disabled={isEditing}
        placeholder="e.g. moderator, content_creator"
      />
      <p class="text-[11px] text-muted-foreground mt-1">Lowercase letters, numbers, and underscores (e.g. store_manager).</p>
    </div>

    <div>
      <label for="role-name-input" class="block text-xs font-medium text-foreground mb-1">Role Display Name</label>
      <Input
        id="role-name-input"
        bind:value={formName}
        placeholder="e.g. Content Moderator"
      />
    </div>

    <div>
      <label for="role-desc-input" class="block text-xs font-medium text-foreground mb-1">Description</label>
      <Input
        id="role-desc-input"
        bind:value={formDesc}
        placeholder="Optional summary of responsibilities"
      />
    </div>

    <!-- Permissions Builder -->
    <div class="pt-2 border-t border-border space-y-3">
      <div class="flex items-center justify-between">
        <h4 class="text-sm font-semibold text-foreground">Permissions</h4>
        <span class="text-xs text-muted-foreground">{formPermissions.length} granted</span>
      </div>

      <div class="flex items-center gap-2">
        <div class="flex-1">
          <Input bind:value={newResource} placeholder="Resource (e.g. collection:posts, media, *)" />
        </div>
        <div class="w-32">
          <Input bind:value={newAction} placeholder="Action (read, create, *)" />
        </div>
        <Button variant="secondary" size="sm" onclick={addPermission}>Add</Button>
      </div>

      <div class="max-h-48 overflow-y-auto space-y-1.5 rounded-lg border border-border p-2 bg-muted/20">
        {#if formPermissions.length === 0}
          <div class="text-xs text-center text-muted-foreground py-3">No permissions assigned yet.</div>
        {:else}
          {#each formPermissions as perm, idx}
            <div class="flex items-center justify-between bg-card px-2.5 py-1.5 rounded border border-border text-xs">
              <span class="font-mono text-foreground font-medium">{perm.resource}</span>
              <div class="flex items-center gap-2">
                <Badge variant="outline">{perm.action}</Badge>
                <button
                  type="button"
                  class="text-muted-foreground hover:text-destructive"
                  onclick={() => removePermission(idx)}
                >
                  ✕
                </button>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    </div>

    <div class="flex justify-end gap-2 pt-4">
      <Button variant="outline" onclick={() => (modalOpen = false)}>Cancel</Button>
      <Button onclick={handleSaveRole} disabled={saving}>
        {saving ? 'Saving...' : 'Save Role'}
      </Button>
    </div>
  </div>
</Modal>

<!-- Confirm Delete Dialog -->
<ConfirmDialog
  open={!!deleteTarget}
  title="Delete Custom Role"
  description={`Are you sure you want to delete role "${deleteTarget?.name}"? Users assigned to this role will lose their custom permissions.`}
  confirmLabel="Delete Role"
  variant="destructive"
  loading={deleting}
  onconfirm={handleDeleteRole}
  oncancel={() => (deleteTarget = null)}
/>
