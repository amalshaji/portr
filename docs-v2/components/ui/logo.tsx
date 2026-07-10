import React from "react";

interface LogoProps {
  width?: number;
  height?: number;
  className?: string;
}

export const Logo: React.FC<LogoProps> = ({
  width = 24,
  height = 24,
  className = "",
}) => (
  <img
    aria-hidden="true"
    alt=""
    className={`portr-logo-mark ${className}`}
    height={height}
    src="/favicon.svg"
    width={width}
  />
);

export default Logo;
