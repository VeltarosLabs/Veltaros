import React, { useEffect } from "react";

type Props = {
    open: boolean;
    title: string;
    children: React.ReactNode;
    onClose: () => void;
};

export default function Modal({ open, title, children, onClose }: Props): React.ReactElement | null {
    useEffect(() => {
        if (!open) return;

        const onKey = (e: KeyboardEvent) => {
            if (e.key === "Escape") onClose();
        };
        window.addEventListener("keydown", onKey);
        return () => window.removeEventListener("keydown", onKey);
    }, [open, onClose]);

    if (!open) return null;

    return (
        <div className="modalOverlay" role="dialog" aria-modal="true" aria-label={title} onMouseDown={onClose}>
            <div className="modal" onMouseDown={(e) => e.stopPropagation()}>
                <div className="modalHeader">
                    <h3 className="modalTitle">{title}</h3>
                    <button className="btn small" type="button" onClick={onClose} aria-label="Close">
                        Close
                    </button>
                </div>
                <div className="modalBody">{children}</div>
            </div>
        </div>
    );
}
