/**
 * Mobile viewport height fix for Safari
 * Handles dynamic viewport changes when keyboard appears/disappears
 */

let lastHeight = window.innerHeight;

export function initViewportFix() {
  // Set initial viewport height
  updateViewportHeight();

  // Update on resize (keyboard show/hide)
  window.addEventListener('resize', handleResize);

  // Update on orientation change
  window.addEventListener('orientationchange', () => {
    setTimeout(updateViewportHeight, 100);
  });

  // iOS specific: update on focus/blur of inputs
  if (isIOS()) {
    document.addEventListener('focusin', handleInputFocus);
    document.addEventListener('focusout', handleInputBlur);
  }
}

function handleResize() {
  const currentHeight = window.innerHeight;

  // Only update if height changed significantly (not just address bar)
  if (Math.abs(currentHeight - lastHeight) > 100) {
    updateViewportHeight();
    lastHeight = currentHeight;
  }
}

function updateViewportHeight() {
  // Set CSS custom property for viewport height
  const vh = window.innerHeight * 0.01;
  document.documentElement.style.setProperty('--vh', `${vh}px`);
}

function handleInputFocus(e: Event) {
  const target = e.target as HTMLElement;
  if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA') {
    // Scroll input into view on iOS
    setTimeout(() => {
      target.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }, 300);
  }
}

function handleInputBlur() {
  // Restore viewport height after keyboard closes
  setTimeout(updateViewportHeight, 300);
}

function isIOS(): boolean {
  return /iPad|iPhone|iPod/.test(navigator.userAgent) && !(window as any).MSStream;
}

export function cleanupViewportFix() {
  window.removeEventListener('resize', handleResize);
  document.removeEventListener('focusin', handleInputFocus);
  document.removeEventListener('focusout', handleInputBlur);
}
