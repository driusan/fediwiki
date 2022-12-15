{ stdenv, buildGoModule }:
buildGoModule rec {
        name = "fediwiki-${version}";
        version = "0.1";

        src = ./.;

        # vendorSha256 = "0000000000000000000000000000000000000000000000000000000000000000";
        # vendorSha256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
        # vendorSha256 = "sha256-pQpattmS9VmO3ZIQUFn66az8GSmB4IvYhTTCFn6SUmo=";
        vendorSha256 = "sha256-3fRHvy6YCUUJE+6iSOMFSqoAkflbWz/axIDAFVv6w7k=";
}
