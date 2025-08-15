use protobuf_codegen::Codegen;

fn main() {
    Codegen::new()
        .protoc()
        .cargo_out_dir("generated_proto")
        .input("src/protos/query.proto")
        .input("src/protos/activate.proto")
        .input("src/protos/subscribe.proto")
        .include("src/protos")
        .run_from_script();
}
