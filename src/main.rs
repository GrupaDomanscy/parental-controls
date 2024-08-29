use std::{io::Read, net::TcpListener, process::exit, time::Duration};

fn parse_request(_: &Vec<u8>) -> bool {
    return true;
}

fn bind_listener() -> Result<(), String> {
    let listener = TcpListener::bind("127.0.0.1:8080")
        .map_err(|err| format!("failed to open socket on port 8080: {}", err))?;

    for stream in listener.incoming() {
        if let Err(e) = &stream {
            println!("failed to open incoming stream: {}", e);
            continue;
        }

        let mut stream = stream.unwrap();

        if let Err(e) = stream.set_read_timeout(Some(Duration::from_secs(15))) {
            println!("failed to set read timeout on incoming stream: {}", e);
            continue;
        }

        let mut buffer: Vec<u8> = Vec::new();

        let mut request_parsed_status = false;

        while !request_parsed_status {
            let _ = stream.read_to_end(&mut buffer);
            request_parsed_status = parse_request(&buffer);
        }

        let stringified_buffer = String::from_utf8(buffer);

        if let Err(e) = &stringified_buffer {
            println!("failed to convert buffer to string: {}", e);
            continue;
        }

        let stringified_buffer = stringified_buffer.unwrap();

        println!("received new response body: {}", stringified_buffer);
    }

    Ok(())
}

fn main() {
    if let Err(e) = bind_listener() {
        println!("{}", e);
        exit(1);
    }
}
