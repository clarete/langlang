#!/usr/bin/env python3
"""
Graph benchmark differences across versions.

Usage:
    python scripts/graph_benches.py --benches-dir <benches_dir> --output-dir <output_dir>

Example:
    python scripts/graph_benches.py --benches-dir go/benches --output-dir /tmp/benchmark_graphs

Arguments:
    --benches-dir <benches_dir>: Directory containing benchmark results (either <dir>/go/<version>/... or <dir>/<version>/...)
    --output-dir <output_dir>: Directory to write generated graphs into
"""

import os
import re
import argparse
from collections import defaultdict
from pathlib import Path
import matplotlib.pyplot as plt
import matplotlib
from typing import Dict, List, Tuple

# Use non-interactive backend for headless environments
matplotlib.use('Agg')


def parse_version(version_str: str) -> Tuple:
    "Parse version string for sorting (e.g., v0.0.10-2-g85ce708)"
    # Extract major.minor.patch and commit count
    match = re.match(r'v(\d+)\.(\d+)\.(\d+)(?:-(\d+))?', version_str)
    if match:
        major, minor, patch, commits = match.groups()
        return (int(major), int(minor), int(patch), int(commits or 0))
    return (0, 0, 0, 0)


def parse_benchmark_file(filepath: Path) -> Dict[str, Dict[str, float]]:
    """Parse a benchmark txt file and extract metrics.

    Returns dict mapping test names to their metrics.
    """
    results = {}
    
    with open(filepath, 'r') as f:
        content = f.read()
    
    # Pattern to match benchmark lines
    # Example: BenchmarkParser/Grammar_csv-10    94896    25213 ns/op    2.58 MB/s    12384 B/op    392 allocs/op
    pattern = re.compile(
        r'(Benchmark\S+)\s+\d+\s+(\d+(?:\.\d+)?)\s+ns/op(?:\s+(\d+(?:\.\d+)?)\s+MB/s)?\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op'
    )
    
    for line in content.split('\n'):
        match = pattern.search(line)
        if match:
            test_name = match.group(1)
            ns_per_op = float(match.group(2))
            mb_per_s = float(match.group(3)) if match.group(3) else None
            bytes_per_op = int(match.group(4))
            allocs_per_op = int(match.group(5))
            
            # Average if we see the same test multiple times
            if test_name not in results:
                results[test_name] = {
                    'ns_per_op': [],
                    'mb_per_s': [],
                    'bytes_per_op': [],
                    'allocs_per_op': []
                }
            
            results[test_name]['ns_per_op'].append(ns_per_op)
            if mb_per_s:
                results[test_name]['mb_per_s'].append(mb_per_s)
            results[test_name]['bytes_per_op'].append(bytes_per_op)
            results[test_name]['allocs_per_op'].append(allocs_per_op)
    
    # Calculate averages
    for test_name in results:
        for metric in results[test_name]:
            if results[test_name][metric]:
                results[test_name][metric] = sum(results[test_name][metric]) / len(results[test_name][metric])
            else:
                results[test_name][metric] = None
    
    return results


def collect_all_benchmarks(benches_dir: Path) -> Dict:
    """Collect all benchmark data from a benches directory.

    The directory layout can be either:
      - benches_dir/go/<version>/<suite>/txt
      - benches_dir/<version>/<suite>/txt   (i.e. benches_dir is already the "go" dir)

    Returns nested dict: version -> suite -> test -> metrics
    """
    data = {}

    go_dir = benches_dir / 'go'
    if not go_dir.exists():
        # Allow passing the "go" directory directly (e.g. repo_root/go/benches)
        go_dir = benches_dir

    if not go_dir.exists():
        print(f"Error: {go_dir} does not exist")
        return data
    
    # Iterate through version directories
    for version_dir in sorted(go_dir.iterdir(), key=lambda x: parse_version(x.name)):
        if not version_dir.is_dir():
            continue
        
        version = version_dir.name
        data[version] = {}
        
        # Iterate through suite directories (charsets, import, json, etc.)
        for suite_dir in version_dir.iterdir():
            if not suite_dir.is_dir():
                continue
            
            suite_name = suite_dir.name
            txt_file = suite_dir / 'txt'
            
            if txt_file.exists():
                data[version][suite_name] = parse_benchmark_file(txt_file)
    
    return data


def plot_metric_over_versions(data: Dict, suite: str, test_pattern: str, metric: str, 
                              output_file: str, title: str, ylabel: str):
    "Plot a specific metric for a test across all versions."
    versions = sorted(data.keys(), key=parse_version)
    
    # Find all tests matching the pattern across all versions
    all_tests = set()
    for version in versions:
        if suite in data[version]:
            for test_name in data[version][suite].keys():
                if test_pattern in test_name:
                    all_tests.add(test_name)
    
    if not all_tests:
        print(f"No tests found matching pattern '{test_pattern}' in suite '{suite}'")
        return
    
    plt.figure(figsize=(14, 8))
    
    for test_name in sorted(all_tests):
        values = []
        version_labels = []
        
        for version in versions:
            if suite in data[version] and test_name in data[version][suite]:
                value = data[version][suite][test_name].get(metric)
                if value is not None:
                    values.append(value)
                    version_labels.append(version)
        
        if values:
            # Simplify test name for legend
            simple_name = test_name.replace('BenchmarkParser/', '').replace('BenchmarkNoCapParser/', 'NoCap/')
            plt.plot(range(len(values)), values, marker='o', label=simple_name, linewidth=2)
    
    plt.xlabel('Version', fontsize=12)
    plt.ylabel(ylabel, fontsize=12)
    plt.title(title, fontsize=14, fontweight='bold')
    plt.legend(bbox_to_anchor=(1.05, 1), loc='upper left', fontsize=9)
    plt.grid(True, alpha=0.3)
    plt.xticks(range(len(version_labels)), version_labels, rotation=45, ha='right', fontsize=8)
    plt.tight_layout()
    plt.savefig(output_file, dpi=150, bbox_inches='tight')
    plt.close()
    print(f"Created: {output_file}")


def create_suite_comparison(data: Dict, suites: List[str], output_dir: Path):
    "Create comparison graphs for each suite across versions."
    output_dir.mkdir(exist_ok=True)
    
    for suite in suites:
        print(f"\nGenerating graphs for suite: {suite}")
        
        # Time per operation (lower is better)
        plot_metric_over_versions(
            data, suite, '', 'ns_per_op',
            str(output_dir / f'{suite}_time.png'),
            f'{suite}: Time per Operation',
            'Nanoseconds per Operation (lower is better)'
        )
        
        # Throughput (higher is better)
        plot_metric_over_versions(
            data, suite, '', 'mb_per_s',
            str(output_dir / f'{suite}_throughput.png'),
            f'{suite}: Throughput',
            'MB/s (higher is better)'
        )
        
        # Memory per operation (lower is better)
        plot_metric_over_versions(
            data, suite, '', 'bytes_per_op',
            str(output_dir / f'{suite}_memory.png'),
            f'{suite}: Memory per Operation',
            'Bytes per Operation (lower is better)'
        )
        
        # Allocations per operation (lower is better)
        plot_metric_over_versions(
            data, suite, '', 'allocs_per_op',
            str(output_dir / f'{suite}_allocs.png'),
            f'{suite}: Allocations per Operation',
            'Allocations per Operation (lower is better)'
        )


def create_overview_graph(data: Dict, output_file: str):
    "Create an overview graph showing average performance across all suites."
    versions = sorted(data.keys(), key=parse_version)
    
    # Calculate average ns/op for each version across all tests
    avg_times = []
    version_labels = []
    
    for version in versions:
        times = []
        for suite in data[version].values():
            for test in suite.values():
                if test.get('ns_per_op'):
                    times.append(test['ns_per_op'])
        
        if times:
            avg_times.append(sum(times) / len(times))
            version_labels.append(version)
    
    # Ensure output directory exists
    Path(output_file).parent.mkdir(parents=True, exist_ok=True)
    
    plt.figure(figsize=(14, 6))
    plt.plot(range(len(avg_times)), avg_times, marker='o', linewidth=2, markersize=8, color='#2E86AB')
    plt.xlabel('Version', fontsize=12)
    plt.ylabel('Average ns/op (lower is better)', fontsize=12)
    plt.title('Overall Performance Trend Across Versions', fontsize=14, fontweight='bold')
    plt.grid(True, alpha=0.3)
    plt.xticks(range(len(version_labels)), version_labels, rotation=45, ha='right', fontsize=8)
    plt.tight_layout()
    plt.savefig(output_file, dpi=150, bbox_inches='tight')
    plt.close()
    print(f"Created: {output_file}")


def print_summary(data: Dict):
    "Print a summary of available benchmarks."
    versions = sorted(data.keys(), key=parse_version)
    print(f"\nFound {len(versions)} versions:")
    print("  " + ", ".join(versions))
    
    all_suites = set()
    for version_data in data.values():
        all_suites.update(version_data.keys())
    
    print(f"\nFound {len(all_suites)} test suites:")
    for suite in sorted(all_suites):
        print(f"  - {suite}")


def main():
    script_dir = Path(__file__).resolve().parent
    repo_root = script_dir.parent

    default_benches_dir = repo_root / 'go' / 'benches'
    if not default_benches_dir.exists():
        default_benches_dir = script_dir / 'benches'

    parser = argparse.ArgumentParser(
        description="Graph benchmark differences across versions",
    )
    parser.add_argument(
        "--benches-dir",
        type=Path,
        default=default_benches_dir,
        help="Directory containing benchmark results (either <dir>/go/<version>/... or <dir>/<version>/...)",
    )
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=(script_dir / "benchmark_graphs"),
        help="Directory to write generated graphs into",
    )
    args = parser.parse_args()

    benches_dir = args.benches_dir.expanduser()
    output_dir = args.output_dir.expanduser()
    
    print("Collecting benchmark data...")
    data = collect_all_benchmarks(benches_dir)
    
    if not data:
        print("No benchmark data found!")
        return
    
    print_summary(data)
    
    # Create overview
    print("\nGenerating overview graph...")
    create_overview_graph(data, str(output_dir / 'overview.png'))
    
    # Get all unique suites
    all_suites = set()
    for version_data in data.values():
        all_suites.update(version_data.keys())
    
    # Create detailed graphs for each suite
    create_suite_comparison(data, sorted(all_suites), output_dir)
    
    print(f"\nAll graphs saved to: {output_dir}")
    print("\nGraph files created:")
    print("  - overview.png: Overall performance trend")
    for suite in sorted(all_suites):
        print(f"  - {suite}_time.png: Time per operation")
        print(f"  - {suite}_throughput.png: Throughput (MB/s)")
        print(f"  - {suite}_memory.png: Memory usage")
        print(f"  - {suite}_allocs.png: Allocations")


if __name__ == '__main__':
    main()