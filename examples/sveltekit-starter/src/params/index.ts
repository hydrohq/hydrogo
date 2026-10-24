// SvelteKit 3 loads a single src/params/index.ts exporting a `params`
// map, instead of scanning the directory for one file per matcher.
import { match as locale } from './locale.ts';

export const params = { locale };
