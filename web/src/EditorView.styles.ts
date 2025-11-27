import { styled } from '@pigment-css/react';

export const RootContainer = styled('div')({
  display: 'grid',
  gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 1fr)',
  gap: '1rem',
  flex: 1,
});

export const Editors = styled('div')({
  display: 'flex',
  flexDirection: 'column',
  gap: '1rem',
  height: '100%',
});

export const EditorContainer = styled('div')({
  flex: 1,
  background: '#1a1a1a',
  border: '2px solid #fbf0df',
  borderRadius: '12px',
  height: '100%',
  display: 'flex',
  flexDirection: 'column',
  overflow: 'hidden',
  '& section': {
    flex: 1,
  },
});

export const ResultContainer = styled('div')({
  // No styles found in original CSS, but defined here for consistency
});

