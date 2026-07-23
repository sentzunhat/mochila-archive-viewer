<script lang="ts">
  import { createEventDispatcher } from "svelte";

  export let authError = "";

  let loginUsername = "";
  let loginFullname = "";

  const dispatch = createEventDispatcher<{ login: { username: string; fullname: string } }>();

  function handleSubmit() {
    if (!loginUsername.trim()) return;
    dispatch("login", { username: loginUsername.trim(), fullname: loginFullname.trim() });
  }
</script>

<main class="min-h-screen flex items-center justify-center bg-archive-bg px-4">
  <section class="w-full max-w-sm bg-archive-panel border border-archive-line rounded-2xl shadow-sm p-8">
    <div class="text-center mb-8">
      <div class="mx-auto mb-4 h-12 w-12 rounded-2xl bg-snapchat border border-snapchat-dark flex items-center justify-center text-2xl">🎒</div>
      <h1 class="text-2xl font-extrabold text-archive-ink m-0 mb-1">Mochila Archive Viewer</h1>
      <p class="text-sm text-archive-muted m-0">Personal data archive explorer</p>
    </div>

    {#if authError}
      <p class="mb-4 text-center text-sm text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2">{authError}</p>
    {/if}

    <form on:submit|preventDefault={handleSubmit} class="flex flex-col gap-4">
      <div>
        <label for="login-username" class="block mb-1 text-sm font-semibold text-archive-ink">Username</label>
        <input
          id="login-username"
          type="text"
          bind:value={loginUsername}
          placeholder="Enter your username"
          autocomplete="username"
          class="w-full px-3 py-2 rounded-lg border border-archive-line bg-white text-archive-ink placeholder:text-archive-muted/70 focus:outline-none focus:ring-2 focus:ring-snapchat-dark"
        />
      </div>
      <div>
        <label for="login-fullname" class="block mb-1 text-sm font-semibold text-archive-ink">Full Name (optional)</label>
        <input
          id="login-fullname"
          type="text"
          bind:value={loginFullname}
          placeholder="Enter your full name"
          autocomplete="name"
          class="w-full px-3 py-2 rounded-lg border border-archive-line bg-white text-archive-ink placeholder:text-archive-muted/70 focus:outline-none focus:ring-2 focus:ring-snapchat-dark"
        />
      </div>
      <button
        type="submit"
        class="mt-2 w-full py-2.5 rounded-lg bg-snapchat text-archive-ink font-extrabold border border-snapchat-dark hover:brightness-95 transition"
      >
        Sign In
      </button>
    </form>
  </section>
</main>
