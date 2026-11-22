import { createKeyValueStore } from '@/lib/browser/keyValueStore';

/** The sample photos of the layout editor, kept per layout in the browser. They never reach the server. */
export const sampleImageStore = createKeyValueStore<Blob>('photopi', 'sample-images');
