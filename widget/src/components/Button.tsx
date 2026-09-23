import type { JSX } from 'preact';

export type ButtonSize = 'normal' | 'small';
export type ButtonVariant = 'filled' | 'clear';
export type ButtonType = 'primary' | 'secondary' | 'cancel' | 'danger';

interface ButtonProps {
  label: string;
  onClick?: (e: Event) => void;
  size?: ButtonSize;
  variant?: ButtonVariant;
  type?: ButtonType;
  htmlType?: 'button' | 'submit' | 'reset';
  disabled?: boolean;
  className?: string;
  /** 비동기 작업이 진행 중임을 스크린 리더에 알릴 때만 지정한다 */
  'aria-busy'?: boolean;
}

export function Button({
  label,
  onClick,
  size = 'normal',
  variant = 'filled',
  type = 'primary',
  htmlType = 'button',
  disabled = false,
  className = '',
  'aria-busy': ariaBusy,
}: ButtonProps): JSX.Element {
  // 기본 클래스
  const baseClass = 'orb-btn';

  // 사이즈 클래스
  const sizeClass = `orb-btn-${size}`;

  // 베리언트 클래스
  const variantClass = `orb-btn-${variant}`;

  // 타입 클래스
  const typeClass = `orb-btn-${type}`;

  const finalClassName = [
    baseClass,
    sizeClass,
    variantClass,
    typeClass,
    className
  ].filter(Boolean).join(' ');

  return (
    <button
      type={htmlType}
      className={finalClassName}
      onClick={onClick}
      disabled={disabled}
      aria-busy={ariaBusy}
    >
      {label}
    </button>
  );
}
