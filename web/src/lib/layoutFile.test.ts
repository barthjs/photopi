import { describe, expect, it } from 'vitest';

import type { LayoutDetailResponse } from '@/api/types';
import { parseLayout } from '@/lib/layoutFile';

/** layout returns a valid layout with one slot and a caption in the default size. */
function layout(): LayoutDetailResponse {
  return {
    id: 'strip',
    name: 'Strip',
    paper: {
      widthMM: 50,
      heightMM: 150,
    },
    pages: [
      {
        slots: [
          {
            x: 0.1,
            y: 0.1,
            w: 0.8,
            h: 0.3,
            locked: true,
          },
        ],
      },
    ],
    branding: {
      enabled: true,
      template: '{{eventName}}',
      x: 0.5,
      y: 0.9,
      rotationDeg: 0,
    },
  };
}

describe('parseLayout', () => {
  it('returns a downloaded layout', () => {
    const downloaded = layout();

    const parsed = parseLayout(JSON.stringify(downloaded));

    expect(parsed).toEqual(downloaded);
  });

  it('keeps the size and color of the caption', () => {
    const downloaded = layout();
    downloaded.branding.fontSizePt = 12;
    downloaded.branding.color = '#ff0000';

    const parsed = parseLayout(JSON.stringify(downloaded));

    expect(parsed?.branding).toMatchObject({
      fontSizePt: 12,
      color: '#ff0000',
    });
  });

  it('reads a layout without ID and name as unnamed', () => {
    const { id: _id, name: _name, ...rest } = layout();

    const parsed = parseLayout(JSON.stringify(rest));

    expect(parsed).toMatchObject({
      id: '',
      name: '',
    });
  });

  it.each([
    {
      name: 'content that is not JSON',
      json: 'not json',
    },
    {
      name: 'content that is not a layout',
      json: '{"name":"layout"}',
    },
    {
      name: 'a layout without pages',
      json: JSON.stringify({
        ...layout(),
        pages: [],
      }),
    },
    {
      name: 'an empty paper',
      json: JSON.stringify({
        ...layout(),
        paper: {
          widthMM: 0,
          heightMM: 150,
        },
      }),
    },
    {
      name: 'a slot outside the page',
      json: JSON.stringify({
        ...layout(),
        pages: [
          {
            slots: [
              {
                x: 1.5,
                y: 0,
                w: 0.5,
                h: 0.5,
                locked: false,
              },
            ],
          },
        ],
      }),
    },
    {
      name: 'a slot without size',
      json: JSON.stringify({
        ...layout(),
        pages: [
          {
            slots: [
              {
                x: 0,
                y: 0,
                w: 0,
                h: 0.5,
                locked: false,
              },
            ],
          },
        ],
      }),
    },
    {
      name: 'a slot without lock',
      json: JSON.stringify({
        ...layout(),
        pages: [
          {
            slots: [
              {
                x: 0,
                y: 0,
                w: 0.5,
                h: 0.5,
              },
            ],
          },
        ],
      }),
    },
    {
      name: 'a layout without caption',
      json: JSON.stringify({
        ...layout(),
        branding: null,
      }),
    },
  ])('rejects $name', ({ json }) => {
    const parsed = parseLayout(json);

    expect(parsed).toBeNull();
  });
});
