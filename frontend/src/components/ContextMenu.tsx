import { useEffect, useRef } from 'react';
import './ContextMenu.css';

interface ContextMenuProps {
  x: number;
  y: number;
  onCopy: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onClose: () => void;
}

export function ContextMenu({ x, y, onCopy, onEdit, onDelete, onClose }: ContextMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        onClose();
      }
    };

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleEscape);

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [onClose]);

  return (
    <div
      ref={menuRef}
      className="context-menu"
      style={{ top: `${y}px`, left: `${x}px` }}
    >
      <button className="context-menu-item" onClick={onCopy}>
        <span className="context-menu-icon">📋</span>
        Copy
      </button>
      <button className="context-menu-item" onClick={onEdit}>
        <span className="context-menu-icon">✏️</span>
        Edit
      </button>
      <button className="context-menu-item context-menu-item-danger" onClick={onDelete}>
        <span className="context-menu-icon">🗑️</span>
        Delete
      </button>
    </div>
  );
}
