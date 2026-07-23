<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { Profile } from "../lib/types";

  export let profile: Profile;
  export let availableUsers: { id: number; username: string; fullName: string }[] = [];
  export let profileUsername = "";
  export let profileFullName = "";

  const dispatch = createEventDispatcher<{
    close: void;
    save: { username: string; fullName: string };
    logout: void;
    switchUser: number;
    newProfile: void;
  }>();
</script>

<div class="modal-backdrop" role="presentation" on:click={() => dispatch("close")}>
  <div class="profile-modal" role="dialog" aria-modal="true" on:click|stopPropagation on:keydown|stopPropagation>
    <div class="profile-head">
      <div>
        <p class="eyebrow">Profile</p>
        <h2>{profile.loggedIn ? "Your archive profile" : "Create a simple profile"}</h2>
      </div>
      <button class="close-button" on:click={() => dispatch("close")} aria-label="Close">×</button>
    </div>

    {#if availableUsers.length > 0}
      <p class="section-label">Switch profile</p>
      <div class="user-list">
        {#each availableUsers as user}
          <button class="user-pill" on:click={() => dispatch("switchUser", user.id)}>
            <span class="user-name">{user.fullName || user.username}</span>
            {#if profile.id === user.id && profile.loggedIn}
              <span class="current-badge">you</span>
            {/if}
          </button>
        {/each}
      </div>
    {/if}

    <label>
      <span>Username</span>
      <input bind:value={profileUsername} placeholder="archivekeeper" />
    </label>
    <label>
      <span>Full name</span>
      <input bind:value={profileFullName} placeholder="Your name" />
    </label>

    <div class="profile-actions">
      <button class="load-more" on:click={() => dispatch("save", { username: profileUsername, fullName: profileFullName })}>
        Save profile
      </button>
      {#if profile.loggedIn}
        <button class="secondary-button" on:click={() => dispatch("newProfile")}>New profile</button>
        <button class="secondary-button" on:click={() => dispatch("logout")}>Logout</button>
      {/if}
    </div>
  </div>
</div>
