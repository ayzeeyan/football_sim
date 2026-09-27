import { describe, expect, test } from 'bun:test';
import React from 'react';
import { ErrorState } from './ui';

describe('ErrorState', () => {
  test('is announced and exposes an optional recovery action', () => {
    const retry = () => undefined;
    const element = ErrorState({ message: 'Feed unavailable', onRetry: retry });
    expect(React.isValidElement(element)).toBe(true);
    expect(element?.props.role).toBe('alert');
    expect(element?.props.children).toBeArray();
  });
});
