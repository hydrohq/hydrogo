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
          errorMessage = json.message || 'Oy kaydedilemedi.';
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
  <div class="p-6 rounded-2xl bg-white border border-zinc-200 shadow-sm space-y-4">
    <div class="flex items-center justify-between border-b border-zinc-200 pb-3">
      <span class="text-xs font-bold text-red-600 uppercase tracking-widest flex items-center gap-1.5">
        <span class="w-2 h-2 rounded-full bg-red-600 animate-ping"></span>
        Günün Anketi
      </span>
      <span class="text-[11px] text-zinc-500 font-medium">{currentTotalVotes.toLocaleString('tr-TR')} Oy</span>
    </div>

    <h3 class="text-base font-bold text-zinc-900 leading-snug">
      {poll.question}
    </h3>

    {#if errorMessage}
      <div class="p-3 rounded-xl bg-red-50 border border-red-200 text-red-700 text-xs font-medium">
        ⚠️ {errorMessage}
      </div>
    {/if}

    {#if !hasVoted}
      <form
        onsubmit={(e) => {
          e.preventDefault();
          submitVote();
        }}
        class="space-y-2.5"
      >
        {#each optionsList as opt}
          <label
            class={`flex items-center gap-3 p-3 rounded-xl border text-xs font-medium cursor-pointer transition-colors ${
              selectedOption === opt
                ? 'bg-red-50 border-red-500 text-red-900 shadow-xs'
                : 'bg-zinc-50/60 border-zinc-200 text-zinc-700 hover:border-zinc-300'
            }`}
          >
            <input
              type="radio"
              name="poll_opt"
              value={opt}
              bind:group={selectedOption}
              disabled={isSubmitting}
              class="w-4 h-4 text-red-600 focus:ring-0 border-zinc-300 cursor-pointer"
            />
            <span class="flex-1">{opt}</span>
          </label>
        {/each}

        <button
          type="submit"
          disabled={!selectedOption || isSubmitting}
          class="w-full mt-3 py-2.5 rounded-xl bg-red-600 hover:bg-red-700 disabled:bg-zinc-200 disabled:text-zinc-500 disabled:cursor-not-allowed text-white font-bold text-xs uppercase tracking-wider transition-colors cursor-pointer shadow-sm flex items-center justify-center gap-2"
        >
          {#if isSubmitting}
            <span class="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin"></span>
            <span>Kaydediliyor...</span>
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
              <span class={`font-medium ${isMyVote ? 'text-red-600 font-bold' : 'text-zinc-700'}`}>
                {opt} {isMyVote ? '✓ (Oyunuz)' : ''}
              </span>
              <span class="font-mono text-zinc-600 font-bold">%{pct}</span>
            </div>
            <div class="w-full h-2 rounded-full bg-zinc-100 overflow-hidden">
              <div
                class={`h-full rounded-full transition-all duration-700 ${
                  isMyVote ? 'bg-red-600' : 'bg-zinc-400'
                }`}
                style={`width: ${pct}%`}
              ></div>
            </div>
          </div>
        {/each}
        <div class="pt-2 text-center text-xs text-emerald-600 font-medium">
          ✓ Oyunuz başarıyla kaydedildi.
        </div>
      </div>
    {/if}
  </div>
{/if}
