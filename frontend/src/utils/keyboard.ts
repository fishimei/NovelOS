import type { KeyboardEvent } from 'react';

export function submitTextareaOnEnter(event: KeyboardEvent<HTMLTextAreaElement>) {
  if (
    event.key !== 'Enter' ||
    event.shiftKey ||
    event.ctrlKey ||
    event.metaKey ||
    event.altKey ||
    event.repeat ||
    event.nativeEvent.isComposing
  ) {
    return;
  }

  const form = event.currentTarget.form;
  const submitter = form?.querySelector<HTMLButtonElement>('button[type="submit"]:not(:disabled)');

  if (!form || !submitter) {
    return;
  }

  event.preventDefault();
  form.requestSubmit(submitter);
}
