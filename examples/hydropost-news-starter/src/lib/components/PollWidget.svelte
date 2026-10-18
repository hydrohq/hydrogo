<script lang="ts">
  import type { Poll } from '#lib';
  import { onMount } from 'svelte';

  interface Props {
    poll: Poll | null;
  }

  let { poll }: Props = $props();

  let selectedOption = $state<string | null>(null);
  let hasVoted = $state(false);
  let isSubmitting = $state(false);
  let errorMessage = $state<string | null>(null);
  let extraVotes = $state(0);
  let currentTotalVotes = $derived((poll?.total_votes || 0) + extraVotes);

  let optionsList = $derived(
    poll?.options
      ? poll.options.split(',').map((o) => o.trim()).filter(Boolean)
      : ['Evet', 'Hayır', 'Kararsızım']
  );

  let votePercentages = $state<Record<string, number>>({});

  function parseExistingVotes() {
    if (!poll?.votes_data) return;
    try {
      const tallies: Record<string, number> = JSON.parse(poll.votes_data);
      const total = poll.total_votes || Object.values(tallies).reduce((a, b) => a + b, 0);
      currentTotalVotes = total;
      const pcts: Record<string, number> = {};
      for (const opt of optionsList) {
        const count = tallies[opt] || 0;
        pcts[opt] = total > 0 ? Math.round((count / total) * 100) : 0;
      }
      votePercentages = pcts;
    } catch {
      // ignore JSON parse error
    }
  }

  async function submitVote() {
    if (!selectedOption || !poll || isSubmitting) return;
    isSubmitting = true;
    errorMessage = null;

    try {
      const res = await fetch('/api/poll/vote', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json',
        },
        body: JSON.stringify({
          pollId: poll.id,
          option: selectedOption,
        }),
      });

      const json = await res.json();
      if (!res.ok) {
        if (json.code === 'already_voted') {
          hasVoted = true;
          parseExistingVotes();
        } else {
          errorMessage = json.message || 'Oy kaydedilemedi, lütfen tekrar deneyin.';
        }
        return;
      }

      // Success
      hasVoted = true;
      extraVotes += 1;
      if (json.percentages) {
        votePercentages = json.percentages;
      }
      if (typeof localStorage !== 'undefined') {
        localStorage.setItem(`voted_poll_${poll.id}`, selectedOption);
      }
    } catch (err) {
      console.error('Failed to submit poll vote:', err);
      errorMessage = 'Sunucuya bağlanılamadı, lütfen tekrar deneyin.';
    } finally {
      isSubmitting = false;
    }
  }

  onMount(() => {
    if (poll) {
      parseExistingVotes();
      if (typeof localStorage !== 'undefined') {
        const saved = localStorage.getItem(`voted_poll_${poll.id}`);
        if (saved) {
          selectedOption = saved;
          hasVoted = true;
        }
      }
    }
  });
</script>

{#if poll}
  <div class="p-5 sm:p-6 rounded-xl bg-[var(--hp-surface)] border border-[var(--hp-border)] space-y-4">
    <div class="flex items-center justify-between">
      <span class="text-xs font-bold text-[var(--hp-primary)] uppercase tracking-widest flex items-center gap-1.5">
        <span class="relative flex w-2 h-2">
          <span class="absolute inline-flex h-full w-full rounded-full bg-[var(--hp-primary)] opacity-60 animate-ping"></span>
          <span class="relative inline-flex rounded-full w-2 h-2 bg-[var(--hp-primary)]"></span>
        </span>
        Günün Anketi
      </span>
      <span class="text-[11px] text-[var(--hp-muted)] font-medium hp-tabular">{currentTotalVotes.toLocaleString('tr-TR')} Oy</span>
    </div>

    <h3 class="text-base font-semibold text-[var(--hp-ink)] leading-snug">
      {poll.question}
    </h3>

    {#if errorMessage}
      <div class="p-3 bg-[var(--hp-primary-light)] border border-[var(--hp-primary-border)] text-[var(--hp-primary-dark)] text-xs font-medium" role="alert">
        {errorMessage}
      </div>
    {/if}

    {#if !hasVoted}
      <form
        onsubmit={(e) => {
          e.preventDefault();
          submitVote();
        }}
        class="space-y-2"
      >
        {#each optionsList as opt}
          <label
            class={`flex items-center gap-3 p-3 rounded-lg border text-xs font-medium cursor-pointer transition-colors ${
              selectedOption === opt
                ? 'bg-[var(--hp-primary-light)] border-[var(--hp-primary)] text-[var(--hp-primary-dark)]'
                : 'bg-[var(--hp-surface)] border-[var(--hp-border)] text-[var(--hp-ink-soft)] hover:border-[var(--hp-border-strong)]'
            }`}
          >
            <input
              type="radio"
              name="poll_opt"
              value={opt}
              bind:group={selectedOption}
              disabled={isSubmitting}
              class="w-4 h-4 accent-[var(--hp-primary)] cursor-pointer"
            />
            <span class="flex-1">{opt}</span>
          </label>
        {/each}

        <button
          type="submit"
          disabled={!selectedOption || isSubmitting}
          class="w-full mt-3 py-2.5 rounded-lg bg-[var(--hp-primary)] hover:bg-[var(--hp-primary-dark)] active:scale-[0.99] disabled:bg-[var(--hp-surface-sunken)] disabled:text-[var(--hp-subtle)] disabled:cursor-not-allowed text-white font-bold text-xs uppercase tracking-wider transition-all cursor-pointer flex items-center justify-center gap-2"
        >
          {#if isSubmitting}
            <span class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin"></span>
            <span>Kaydediliyor…</span>
          {:else}
            <span>Oy Kullan</span>
          {/if}
        </button>
      </form>
    {:else}
      <!-- Results view with real percentage bars from database -->
      <div class="space-y-3 pt-1">
        {#each optionsList as opt}
          {@const pct = votePercentages[opt] ?? 0}
          {@const isMyVote = selectedOption === opt}
          <div class="space-y-1">
            <div class="flex items-center justify-between text-xs">
              <span class={`font-medium ${isMyVote ? 'text-[var(--hp-primary)] font-bold' : 'text-[var(--hp-ink-soft)]'}`}>
                {opt} {isMyVote ? '— oyunuz' : ''}
              </span>
              <span class="hp-tabular text-[var(--hp-muted)] font-bold">%{pct}</span>
            </div>
            <div class="w-full h-1.5 rounded-full bg-[var(--hp-surface-sunken)] overflow-hidden">
              <div
                class={`h-full rounded-full transition-all duration-700 ${
                  isMyVote ? 'bg-[var(--hp-primary)]' : 'bg-[var(--hp-border-strong)]'
                }`}
                style={`width: ${pct}%`}
              ></div>
            </div>
          </div>
        {/each}
        <div class="pt-2 text-center text-xs text-emerald-700 font-medium">
          Oyunuz kaydedildi.
        </div>
      </div>
    {/if}
  </div>
{/if}
