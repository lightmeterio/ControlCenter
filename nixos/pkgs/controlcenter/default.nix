{ stdenv, buildGoPackage, fetchgit}:

buildGoPackage rec {
  pname = "lightmeter";
  version = "0.0.6-1";
  rev = "1e5fb1e422a1fd0d6bb54c14396671968366851a";

  goPackagePath = "gitlab.com/lightmeter/controlcenter";

  src = fetchgit {
     inherit rev;
     url = "https://gitlab.com/lightmeter/controlcenter";
     sha256 = "104hws6p6kbrg361vq5ma19838hvabb5gldibyx55aqbijrg5xw0";
  };

  goDeps = ./deps.nix;
}