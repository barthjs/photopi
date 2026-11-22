// On-screen keyboard layouts for a touchscreen.
// The email layout leaves out the keys an address does not need.

export type KeyboardKey =
  | { kind: 'char'; label: string; span: number }
  | { kind: 'backspace'; span: number }
  | { kind: 'shift'; span: number }
  | { kind: 'symbols'; span: number }
  | { kind: 'space'; span: number }
  | { kind: 'spacer'; span: number };

export type KeyboardLayout = KeyboardKey[][];

export type KeyboardLanguage = 'en' | 'de';

/** Turns a string into one key per character. */
function chars(letters: string): KeyboardKey[] {
  return letters.split('').map((label) => ({
    kind: 'char',
    label,
    span: 1,
  }));
}

export const keyboardLayouts: Record<KeyboardLanguage, KeyboardLayout> = {
  en: [
    [
      ...chars('1234567890-'),
      {
        kind: 'backspace',
        span: 1,
      },
    ],
    [
      ...chars('qwertyuiop_'),
      {
        kind: 'spacer',
        span: 1,
      },
    ],
    [
      ...chars('asdfghjkl'),
      {
        kind: 'char',
        label: '@',
        span: 3,
      },
    ],
    [
      ...chars('zxcvbnm'),
      {
        kind: 'char',
        label: '.',
        span: 5,
      },
    ],
  ],
  de: [
    [
      ...chars('1234567890-'),
      {
        kind: 'backspace',
        span: 1,
      },
    ],
    [...chars('qwertzuiopü_')],
    [
      ...chars('asdfghjklöä'),
      {
        kind: 'spacer',
        span: 1,
      },
    ],
    [
      ...chars('yxcvbnmß'),
      {
        kind: 'char',
        label: '@',
        span: 2,
      },
      {
        kind: 'char',
        label: '.',
        span: 2,
      },
    ],
  ],
};

/** Returns the last row of the text layouts. */
function bottomRow(): KeyboardKey[] {
  return [
    {
      kind: 'symbols',
      span: 2,
    },
    {
      kind: 'space',
      span: 8,
    },
    {
      kind: 'backspace',
      span: 2,
    },
  ];
}

/** The letters of the text layouts. The shift key turns them into capital letters. */
export const textLayouts: Record<KeyboardLanguage, KeyboardLayout> = {
  en: [
    [...chars('1234567890-=')],
    [...chars('qwertyuiop[]')],
    [
      ...chars("asdfghjkl;'"),
      {
        kind: 'spacer',
        span: 1,
      },
    ],
    [
      {
        kind: 'shift',
        span: 2,
      },
      ...chars('zxcvbnm,./'),
    ],
    bottomRow(),
  ],
  de: [
    [...chars('1234567890ß-')],
    [...chars('qwertzuiopü+')],
    [...chars('asdfghjklöä#')],
    [
      {
        kind: 'shift',
        span: 2,
      },
      ...chars('yxcvbnm,.-'),
    ],
    bottomRow(),
  ],
};

/** The symbol page of the text layouts, the same in every language. */
export const symbolLayout: KeyboardLayout = [
  [...chars('!@#$%^&*()_+')],
  [...chars('~`{}[]|\\<>/?')],
  [...chars('"\':;=,.€§°²³')],
  [
    {
      kind: 'spacer',
      span: 2,
    },
    ...chars('1234567890'),
  ],
  bottomRow(),
];
