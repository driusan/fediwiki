{ stdenv, buildGoModule }:
buildGoModule rec {
        name = "fediwiki-${version}";
        version = "0.1";

        src = ./.;

        # vendorSha256 = "0000000000000000000000000000000000000000000000000000000000000000";
        vendorSha256 = "sha256-bDihSeXuWVa311tvOqfCpBxYevJCAWaZd+m7GQeA6i0=";
}
