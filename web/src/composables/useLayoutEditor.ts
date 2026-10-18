import { ref } from 'vue';

import type { LayoutDetailResponse } from '@/api/types';
import { useBackgroundImage } from '@/composables/editor/useBackgroundImage';
import { useEditorGuides } from '@/composables/editor/useEditorGuides';
import { useEditorSelection } from '@/composables/editor/useEditorSelection';
import { useEditorVisibility } from '@/composables/editor/useEditorVisibility';
import { useHistory } from '@/composables/useHistory';
import { useRatioLock } from '@/composables/useRatioLock';
import { sampleImageStore } from '@/lib/sampleImages';

/**
 * useLayoutEditor holds the editing state of the layout canvas, such as the selection, the hidden slots, the history,
 * the sample photo, the guides, and the ratio lock of the selected slot. It stays apart from the layout itself,
 * which holds only the data that is saved. The selection holds the indices of the slots of the active page.
 */
export function useLayoutEditor(
  state: LayoutDetailResponse,
  slotIndices: () => readonly number[],
  savedId: () => string | null,
) {
  const selection = useEditorSelection(slotIndices);

  // The slots hidden for a while, such as to reach one that another covers
  const visibility = useEditorVisibility<number>();

  // Whether slots are dragged, which holds back the automatic history checkpoint until the drag ends
  const dragging = ref(false);

  const history = useHistory(state, {
    paused: () => dragging.value,
  });

  // A sample photo the slots show like the print, cropped from its center to the shape of each slot
  const sampleImage = useBackgroundImage({
    store: sampleImageStore,
    keyPrefix: 'layout',
    storageKey: 'layoutEditor',
  });

  // The lines dragged out of the rulers, which the browser keeps per saved layout
  const guides = useEditorGuides({
    storageKey: 'layoutEditor',
    document: savedId,
  });

  // The ratio lock of the selected slot is an editing aid and not saved
  const slotRatioLock = useRatioLock();

  return {
    selection,
    visibility,
    dragging,
    history,
    sampleImage,
    guides,
    slotRatioLock,
  };
}
