{ pkgs, lib, config, ... }:
{
  packages = [
    pkgs.pkg-config
    pkgs.libcap
    pkgs.gcc
  ];
  languages = {
    go.enable = true;
  };
}

