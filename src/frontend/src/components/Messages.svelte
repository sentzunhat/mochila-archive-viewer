<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { Conversation, ChatMessage } from "../lib/types";
  import { formatter, messageTypeLabel } from "../utils/formatters";
  import { mediaHttpSrc } from "../utils/mediaUrl";

  export let conversations: Conversation[] = [];
  export let selectedConversation: Conversation | null = null;
  export let loadingConversation = false;
  export let activePlatform = "snapchat";
  export let profileId = 0;

  const dispatch = createEventDispatcher<{
    selectConversation: string;
    selectMessageMedia: number;
  }>();
</script>

<section class="messages-layout">
  <aside class="conversation-list">
    {#each conversations as conversation}
      <button
        class:active={selectedConversation?.id === conversation.id}
        on:click={() => dispatch("selectConversation", conversation.id)}
      >
        <strong>{conversation.title}</strong>
        <span>
          {formatter.format(conversation.messageCount)} messages ·
          {formatter.format(conversation.savedCount)} saved ·
          {conversation.lastCreated ?? "no date"}
        </span>
      </button>
    {/each}
  </aside>

  <section class="message-panel">
    {#if loadingConversation}
      <div class="empty">Loading conversation...</div>
    {:else if !selectedConversation}
      <div class="empty">No chat history found in this export.</div>
    {:else}
      <div class="message-heading">
        <div>
          <h2>{selectedConversation.title}</h2>
          <p>
            {formatter.format(selectedConversation.messageCount)} messages ·
            {formatter.format(selectedConversation.mediaCount)} media references
          </p>
        </div>
      </div>

      <div class="messages">
        {#each selectedConversation.messages ?? [] as message}
          <article class:sent={message.isSender} class="message">
            <div>
              <strong>{message.from || (message.isSender ? "You" : selectedConversation.title)}</strong>
              <span>{message.created}</span>
            </div>
            {#if message.content}
              <p>{message.content}</p>
            {:else if !message.mediaId}
              <p class="italic text-archive-muted">{messageTypeLabel(message.mediaType)}</p>
            {/if}
            {#if message.mediaId != null}
              <button
                class="mt-2 block w-full max-w-[220px] overflow-hidden rounded-lg border border-archive-line bg-archive-bg"
                on:click={() => dispatch("selectMessageMedia", message.mediaId)}
                aria-label="Open attached media"
              >
                {#if message.linkedMediaType === "video"}
                  <video preload="metadata" muted class="block max-h-56 w-full object-cover" src={mediaHttpSrc(activePlatform, profileId, message.mediaId)}></video>
                {:else}
                  <img loading="lazy" class="block max-h-56 w-full object-cover" src={mediaHttpSrc(activePlatform, profileId, message.mediaId)} alt="Attached media" />
                {/if}
              </button>
            {/if}
            {#if message.isSaved}
              <small>saved</small>
            {/if}
          </article>
        {/each}
      </div>
    {/if}
  </section>
</section>
