import { cubicOut } from 'svelte/easing';

/** Every duration collapses to 0 for visitors who ask the OS for reduced motion. */
const reduce = typeof matchMedia !== 'undefined' && matchMedia('(prefers-reduced-motion: reduce)').matches;
export const ms = (n: number) => (reduce ? 0 : n);
export const ease = cubicOut;
