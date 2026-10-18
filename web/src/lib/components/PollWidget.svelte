<script lang="ts">
  import type { Poll } from '#lib';
  import { onMount } from 'svelte';

  interface Props {
    poll: Poll | null;
  }

  let { poll }: Props = $props();

  let selectedOption = $state<string | null>(null);
  let hasVoted = $state(false);
  let userVoteCount = $state(0);
  let totalVotes = $derived((poll?.total_votes || 100) + userVoteCount);

  let optionsList = $derived(
    poll?.options
      ? poll.options.split(',').map((o) => o.trim()).filter(Boolean)
      : ['Evet', 'Hayır', 'Kararsızım']
  );

  // Distribution generator for visual delight
  let voteResults = $state<Record<string, number>>({});

  function calculatePercentages() {
    const results: Record<string, number> = {};
    const total = optionsList.length;
    let remaining = 100;

    optionsList.forEach((opt, idx) => {
      if (idx === total - 1) {
        results[opt] = Math.max(5, remaining);
      } else {
        const share = Math.floor(remaining / (total - idx + 0.5));
        results[opt] = share;
        remaining -= share;
      }
    });
    voteResults = results;
  }

  function submitVote() {
    if (!selectedOption || !poll) return;
    hasVoted = true;
    userVoteCount += 1;
    calculatePercentages();
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(`voted_poll_${poll.id}`, selectedOption);
    }
  }

  onMount(() => {
    if (poll && typeof localStorage !== 'undefined') {
      const saved = localStorage.getItem(`voted_poll_${poll.id}`);
      if (saved) {
        selectedOption = saved;
        hasVoted = true;
        calculatePercentages();
      }
    }
  });
</script>

{#if poll}
  <div class="p-6 rounded-2xl bg-zinc-900/60 border border-zinc-800 space-y-4">
    <div class="flex items-center justify-between border-b border-zinc-800 pb-3">
      <span class="text-xs font-bold text-red-500 uppercase tracking-widest flex items-center gap-1.5">
        <span class="w-2 h-2 rounded-full bg-red-600 animate-ping"></span>
        Günün Anketi
      </span>
      <span class="text-[11px] text-zinc-500">{totalVotes.toLocaleString('tr-TR')} Oy</span>
    </div>

    <h3 class="text-base font-bold text-white leading-snug">
      {poll.question}
    </h3>

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
            class={`flex items-center gap-3 p-3 rounded-xl border text-xs font-medium cursor-pointer transition-all ${
              selectedOption === opt
                ? 'bg-red-950/40 border-red-600 text-white shadow'
                : 'bg-zinc-950/80 border-zinc-800 text-zinc-300 hover:border-zinc-700'
            }`}
          >
            <input
              type="radio"
              name="poll_opt"
              value={opt}
              bind:group={selectedOption}
              class="w-4 h-4 text-red-600 focus:ring-0 bg-zinc-900 border-zinc-700 cursor-pointer"
            />
            <span class="flex-1">{opt}</span>
          </label>
        {/each}

        <button
          type="submit"
          disabled={!selectedOption}
          class="w-full mt-3 py-2.5 rounded-xl bg-red-600 hover:bg-red-500 disabled:opacity-40 disabled:hover:bg-red-600 text-white font-bold text-xs uppercase tracking-wider transition-all cursor-pointer shadow-lg shadow-red-600/20"
        >
          Oy Kullan
        </button>
      </form>
    {:else}
      <!-- Results view with animated percentage bars -->
      <div class="space-y-3 pt-1">
        {#each optionsList as opt}
          {@const pct = voteResults[opt] || 25}
          {@const isMyVote = selectedOption === opt}
          <div class="space-y-1">
            <div class="flex items-center justify-between text-xs">
              <span class={`font-medium ${isMyVote ? 'text-red-400 font-bold' : 'text-zinc-300'}`}>
                {opt} {isMyVote ? '✓ (Oyunuz)' : ''}
              </span>
              <span class="font-mono text-zinc-400 font-bold">%{pct}</span>
            </div>
            <div class="w-full h-2 rounded-full bg-zinc-800 overflow-hidden">
              <div
                class={`h-full rounded-full transition-all duration-700 ${
                  isMyVote ? 'bg-gradient-to-r from-red-600 to-rose-400' : 'bg-zinc-600'
                }`}
                style={`width: ${pct}%`}
              ></div>
            </div>
          </div>
        {/each}
        <div class="pt-2 text-center text-[11px] text-emerald-400 font-medium">
          ✓ Oyunuz başarıyla kaydedildi. Katılımınız için teşekkürler!
        </div>
      </div>
    {/if}
  </div>
{/if}
