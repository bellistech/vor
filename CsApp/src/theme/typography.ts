// bellis.tech design system — typography
export const typography = {
  fontBody: 'SpaceGrotesk-Regular',
  fontBodyMedium: 'SpaceGrotesk-Medium',
  fontBodySemiBold: 'SpaceGrotesk-SemiBold',
  fontMono: 'JetBrainsMono-Regular',

  heading: {
    fontFamily: 'SpaceGrotesk-SemiBold',
    fontSize: 24,
    lineHeight: 32,
  },
  subheading: {
    fontFamily: 'SpaceGrotesk-Medium',
    fontSize: 18,
    lineHeight: 24,
  },
  body: {
    fontFamily: 'SpaceGrotesk-Regular',
    fontSize: 16,
    lineHeight: 24,
  },
  code: {
    fontFamily: 'JetBrainsMono-Regular',
    fontSize: 14,
    lineHeight: 20,
  },
  label: {
    fontFamily: 'JetBrainsMono-Regular',
    fontSize: 12,
    lineHeight: 16,
    textTransform: 'uppercase' as const,
    letterSpacing: 0.5,
  },
  meta: {
    fontFamily: 'JetBrainsMono-Regular',
    fontSize: 11,
    lineHeight: 14,
  },
} as const;
