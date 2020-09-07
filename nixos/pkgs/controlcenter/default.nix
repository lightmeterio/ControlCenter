{ buildGoModule, fetchgit, lib }:

buildGoModule rec {
  pname = "lightmeter";
  version = "master";
  rev = "dad4451218d3fb99bc1d3240f66dac030ee8f246";

  deleteVendor = false;

  goPackagePath = "gitlab.com/lightmeter/controlcenter";

  src = fetchgit {
     inherit rev;
     url = "https://gitlab.com/lightmeter/controlcenter";
     sha256 = "1pgkf9zlhkblzdjipp17zmpp9di19hizpkqav6bk9i6ajnmf6am8";
  };

  vendorSha256 = null;

  checkPhase = ''
  '';

  buildPhase = ''
  go generate -tags="release" ./staticdata
  go generate ./domainmapping
  go generate ./po
  go install -tags="release"
  '';

  installPhase =  ''
      runHook preInstall
      mkdir -p $out
      dir="$GOPATH/bin"
      [ -e "$dir" ] && cp -r $dir $out
      runHook postInstall
  '';

  meta = with lib; {
    description = "Lightmeter Control Center, the Open Source mailtech monitoring and management application.";
    homepage = https://lightmeter.io/about/;

    maintainers = with maintainers; [ donutloop ];
    platforms = platforms.linux ++ platforms.darwin;
  };
}