
# DorkingZ

![DorkingZ Banner](https://github.com/sultanzio/dorkingz/raw/main/banner.png)

**DorkingZ** is a high-performance Go-based tool designed for automated search engine querying using custom search dorks and rotating proxies. It efficiently retrieves unique domains from search results across multiple search engines, including Google, Bing, and DuckDuckGo.

## Features

- **Multiple Search Engines**: Supports Google, Bing, and DuckDuckGo.
- **Rotating Proxies**: Utilizes a pool of validated proxies to prevent IP blocking and bypass rate limits.
- **Concurrent Processing**: Leverages Go's concurrency model to perform multiple searches simultaneously.
- **Retry Mechanism**: Implements exponential backoff retries with alternate proxies upon failures.
- **Unique Domain Extraction**: Parses search results to extract and store unique domain names.
- **Configurable Parameters**: Easily configurable through command-line flags for dorks, proxies, concurrency levels, and more.
- **Efficient Logging**: Provides informative logs to monitor the progress and status of searches.

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
  - [Dork File (`dork.txt`)](#dork-file-dorktxt)
  - [Proxy File (`proxy.txt`)](#proxy-file-proxytxt)
- [Command-Line Flags](#command-line-flags)
- [Example](#example)
- [Contributing](#contributing)
- [License](#license)
- [Contact](#contact)

## Installation

### Prerequisites

- **Go**: Ensure you have Go installed on your system. You can download it from [golang.org](https://golang.org/dl/).

### Clone the Repository

```bash
git clone https://github.com/sultanzio/dorkingz.git
cd dorkingz
```

### Build the Project

```bash
go build -o dorkingz main.go
```

This command compiles the Go source code and produces an executable named `dorkingz`.

## Usage

After building the project, you can run `dorkingz` using the command line with various flags to customize its behavior.

```bash
./dorkingz -d dork.txt -p 5 -o results.txt -e google,bing,duckduckgo -t 500 -x proxy.txt -r 3
```

### Command-Line Flags

- `-d`: **(Required)** Path to the `dork.txt` file containing multiple dorks (search queries).
- `-p`: **(Optional)** Number of pages to search per dork. *(Default: 1)*
- `-o`: **(Optional)** Output file to save the results. *(Default: results.txt)*
- `-e`: **(Optional)** Comma-separated list of search engines to use. Options: `google`, `bing`, `duckduckgo`. *(Default: google)*
- `-t`: **(Optional)** Number of concurrent requests. *(Default: 500)*
- `-x`: **(Optional)** Path to the `proxy.txt` file containing a list of proxies. *(Default: proxy.txt)*
- `-r`: **(Optional)** Maximum number of retries per request. *(Default: 3)*

## Configuration

### Dork File (`dork.txt`)

The `dork.txt` file should contain one dork per line. A dork is a specific search query that targets particular information on the web.

**Example `dork.txt`:**

```
site:example.com inurl:admin
intitle:"index of" "parent directory"
filetype:sql "password"
```

### Proxy File (`proxy.txt`)

The `proxy.txt` file should list proxies in the format `ip:port`. If your proxies require authentication, use the format `username:password@ip:port`. You can use premium proxy or https://www.sslproxies.org/ free!

**Example `proxy.txt`:**

```
192.168.1.100:8080
user1:pass1@192.168.1.101:8080
user2:pass2@192.168.1.102:8080
```

## Command-Line Flags

Below is a detailed explanation of each command-line flag:

| Flag | Description | Default |
|------|-------------|---------|
| `-d` | **(Required)** Path to the `dork.txt` file containing search queries. | N/A |
| `-p` | Number of pages to search per dork. Each page typically contains 10 results. | `1` |
| `-o` | Output file to save the extracted domains. | `results.txt` |
| `-e` | Comma-separated list of search engines to use. Options: `google`, `bing`, `duckduckgo`. | `google` |
| `-t` | Number of concurrent requests to process. Higher values increase speed but require more system resources. | `500` |
| `-x` | Path to the `proxy.txt` file containing proxy addresses. | `proxy.txt` |
| `-r` | Maximum number of retries per failed request. | `3` |

## Example

Assuming you have `dork.txt` and `proxy.txt` properly configured, here's how you can run `dorkingz`:

```bash
./dorkingz -d dork.txt -p 3 -o unique_domains.txt -e google,bing -t 300 -x proxy.txt -r 5
```

This command will:

- Use the search queries from `dork.txt`.
- Search 3 pages per dork.
- Save the unique domains to `unique_domains.txt`.
- Utilize both Google and Bing as search engines.
- Run up to 300 concurrent requests.
- Use proxies listed in `proxy.txt`.
- Retry failed requests up to 5 times.

## Contributing

Contributions are welcome! Please follow these steps:

1. **Fork the Repository**: Click the [Fork](https://github.com/sultanzio/dorkingz/fork) button on the top-right corner of this page.
2. **Clone Your Fork**:

    ```bash
    git clone https://github.com/your-username/dorkingz.git
    cd dorkingz
    ```

3. **Create a New Branch**:

    ```bash
    git checkout -b feature/YourFeatureName
    ```

4. **Make Your Changes**: Implement your feature or bug fix.
5. **Commit Your Changes**:

    ```bash
    git commit -m "Add your message here"
    ```

6. **Push to Your Fork**:

    ```bash
    git push origin feature/YourFeatureName
    ```

7. **Open a Pull Request**: Navigate to the original repository and open a pull request detailing your changes.

## License

Distributed under the MIT License. See `LICENSE` for more information.

## Contact

Project Link: [https://github.com/sultanzio/dorkingz](https://github.com/sultanzio/dorkingz)

Author: [SultanZio](https://github.com/sultanzio)

Feel free to reach out for any questions or support!

## Disclaimer

**DorkingZ** is provided for educational and ethical purposes only. The author does not condone or support any illegal activities, including but not limited to unauthorized access to computer systems, data breaches, or any form of cyber exploitation.

**Users are solely responsible** for ensuring that their use of **DorkingZ** complies with all applicable laws, regulations, and policies. The author and contributors shall not be held liable for any misuse or damage resulting from the use of this tool.

Before using **DorkingZ**, please obtain proper authorization and consent from the relevant parties. Misuse of this tool for malicious purposes is strictly prohibited and may result in legal consequences.

By using **DorkingZ**, you agree to these terms and acknowledge that you understand and accept full responsibility for your actions.

---

**Legal Notice:** This disclaimer is intended to protect the author and contributors from legal liability. However, it does not replace professional legal advice. For specific legal concerns, please consult with a qualified attorney.

