#!/usr/bin/env python3
"""
Comprehensive test runner for Portr project
Runs tests for admin, server, and CLI components
"""

import os
import sys
import subprocess
import argparse
from pathlib import Path
from typing import List, Tuple


class Colors:
    GREEN = '\033[92m'
    RED = '\033[91m'
    YELLOW = '\033[93m'
    BLUE = '\033[94m'
    BOLD = '\033[1m'
    END = '\033[0m'


def print_colored(message: str, color: str = Colors.END):
    print(f"{color}{message}{Colors.END}")


def print_header(message: str):
    print_colored(f"\n{'='*60}", Colors.BLUE)
    print_colored(f"{message}", Colors.BOLD)
    print_colored(f"{'='*60}", Colors.BLUE)


def run_command(command: List[str], cwd: str = None) -> Tuple[bool, str, str]:
    """Run a command and return success status, stdout, stderr"""
    try:
        result = subprocess.run(
            command,
            cwd=cwd,
            capture_output=True,
            text=True,
            timeout=300  # 5 minutes timeout
        )
        return result.returncode == 0, result.stdout, result.stderr
    except subprocess.TimeoutExpired:
        return False, "", "Command timed out after 5 minutes"
    except Exception as e:
        return False, "", str(e)


def check_dependencies():
    """Check if required dependencies are available"""
    print_header("Checking Dependencies")
    
    dependencies = [
        ("python", ["python3", "--version"]),
        ("go", ["go", "version"]),
        ("pytest", ["python3", "-c", "import pytest; print(f'pytest {pytest.__version__}')"]),
        ("tortoise", ["python3", "-c", "import tortoise; print('tortoise-orm available')"]),
    ]
    
    missing = []
    for name, cmd in dependencies:
        success, stdout, stderr = run_command(cmd)
        if success:
            print_colored(f"✓ {name}: {stdout.strip()}", Colors.GREEN)
        else:
            print_colored(f"✗ {name}: Not available", Colors.RED)
            missing.append(name)
    
    if missing:
        print_colored(f"\nMissing dependencies: {', '.join(missing)}", Colors.RED)
        print_colored("Please install missing dependencies before running tests", Colors.YELLOW)
        return False
    
    return True


def run_admin_tests(test_filter: str = ""):
    """Run Python admin tests"""
    print_header("Running Admin Tests (Python)")
    
    admin_dir = Path("admin")
    if not admin_dir.exists():
        print_colored("Admin directory not found", Colors.RED)
        return False
    
    os.chdir(admin_dir)
    
    # Build pytest command
    cmd = ["python3", "-m", "pytest"]
    
    if test_filter:
        cmd.extend(["-k", test_filter])
    
    # Add common pytest options
    cmd.extend([
        "-v",  # verbose
        "--tb=short",  # shorter tracebacks
        "--disable-warnings",  # reduce noise
        "tests/",
    ])
    
    print_colored(f"Running: {' '.join(cmd)}", Colors.BLUE)
    success, stdout, stderr = run_command(cmd)
    
    if success:
        print_colored("✓ Admin tests passed", Colors.GREEN)
        print(stdout)
        return True
    else:
        print_colored("✗ Admin tests failed", Colors.RED)
        print(stderr)
        print(stdout)
        return False


def run_server_tests(test_filter: str = ""):
    """Run Go server tests"""
    print_header("Running Server Tests (Go)")
    
    tunnel_dir = Path("tunnel")
    if not tunnel_dir.exists():
        print_colored("Tunnel directory not found", Colors.RED)
        return False
    
    os.chdir(tunnel_dir)
    
    # Run tests for different server packages
    test_packages = [
        "./internal/server/config",
        "./internal/server/proxy", 
        "./internal/server/service",
        "./internal/server/cron",
        "./internal/utils",
    ]
    
    all_success = True
    
    for package in test_packages:
        print_colored(f"\nTesting package: {package}", Colors.BLUE)
        
        cmd = ["go", "test", "-v"]
        if test_filter:
            cmd.extend(["-run", test_filter])
        cmd.append(package)
        
        success, stdout, stderr = run_command(cmd)
        
        if success:
            print_colored(f"✓ {package} tests passed", Colors.GREEN)
            if stdout:
                print(stdout)
        else:
            print_colored(f"✗ {package} tests failed", Colors.RED)
            if stderr:
                print(stderr)
            if stdout:
                print(stdout)
            all_success = False
    
    return all_success


def run_cli_tests(test_filter: str = ""):
    """Run Go CLI tests"""
    print_header("Running CLI Tests (Go)")
    
    tunnel_dir = Path("tunnel")
    if not tunnel_dir.exists():
        print_colored("Tunnel directory not found", Colors.RED)
        return False
    
    os.chdir(tunnel_dir)
    
    # Run tests for CLI packages
    test_packages = [
        "./cmd/portr",
        "./internal/client/config",
        "./internal/client/client",
    ]
    
    all_success = True
    
    for package in test_packages:
        print_colored(f"\nTesting package: {package}", Colors.BLUE)
        
        cmd = ["go", "test", "-v", "-short"]  # Use -short to skip integration tests
        if test_filter:
            cmd.extend(["-run", test_filter])
        cmd.append(package)
        
        success, stdout, stderr = run_command(cmd)
        
        if success:
            print_colored(f"✓ {package} tests passed", Colors.GREEN)
            if stdout:
                print(stdout)
        else:
            print_colored(f"✗ {package} tests failed", Colors.RED)
            if stderr:
                print(stderr)
            if stdout:
                print(stdout)
            all_success = False
    
    return all_success


def run_integration_tests():
    """Run integration tests across components"""
    print_header("Running Integration Tests")
    
    # These would be tests that verify the entire system works together
    # For now, we'll run a subset of admin integration tests
    
    admin_dir = Path("admin")
    if not admin_dir.exists():
        print_colored("Admin directory not found", Colors.RED)
        return False
    
    os.chdir(admin_dir)
    
    cmd = [
        "python3", "-m", "pytest",
        "-v",
        "-k", "integration",
        "--tb=short",
        "tests/",
    ]
    
    print_colored(f"Running: {' '.join(cmd)}", Colors.BLUE)
    success, stdout, stderr = run_command(cmd)
    
    if success:
        print_colored("✓ Integration tests passed", Colors.GREEN)
        if stdout:
            print(stdout)
        return True
    else:
        print_colored("✗ Integration tests failed", Colors.RED)
        if stderr:
            print(stderr)
        if stdout:
            print(stdout)
        return False


def run_linting():
    """Run linting for all components"""
    print_header("Running Linting")
    
    results = []
    original_cwd = os.getcwd()
    
    # Python linting (admin)
    admin_dir = Path("admin")
    if admin_dir.exists():
        os.chdir(admin_dir)
        
        # Check if flake8 is available
        success, _, _ = run_command(["python3", "-c", "import flake8"])
        if success:
            print_colored("Running flake8 on admin code...", Colors.BLUE)
            success, stdout, stderr = run_command([
                "python3", "-m", "flake8", 
                "--max-line-length=88",
                "--ignore=E203,W503",
                "."
            ])
            if success:
                print_colored("✓ Admin linting passed", Colors.GREEN)
            else:
                print_colored("✗ Admin linting failed", Colors.RED)
                print(stdout)
            results.append(("Admin linting", success))
        else:
            print_colored("Skipping flake8 (not installed)", Colors.YELLOW)
        
        os.chdir(original_cwd)
    
    # Go linting (server & CLI)
    tunnel_dir = Path("tunnel")
    if tunnel_dir.exists():
        os.chdir(tunnel_dir)
        
        # Go fmt
        print_colored("Running go fmt...", Colors.BLUE)
        success, stdout, stderr = run_command(["go", "fmt", "./..."])
        if success:
            print_colored("✓ Go formatting passed", Colors.GREEN)
        else:
            print_colored("✗ Go formatting failed", Colors.RED)
            print(stderr)
        results.append(("Go formatting", success))
        
        # Go vet
        print_colored("Running go vet...", Colors.BLUE)
        success, stdout, stderr = run_command(["go", "vet", "./..."])
        if success:
            print_colored("✓ Go vet passed", Colors.GREEN)
        else:
            print_colored("✗ Go vet failed", Colors.RED)
            print(stderr)
        results.append(("Go vet", success))
        
        os.chdir(original_cwd)
    
    return all(result[1] for result in results)


def generate_coverage_report():
    """Generate test coverage reports"""
    print_header("Generating Coverage Reports")
    
    original_cwd = os.getcwd()
    
    # Python coverage (admin)
    admin_dir = Path("admin")
    if admin_dir.exists():
        os.chdir(admin_dir)
        
        # Check if coverage is available
        success, _, _ = run_command(["python3", "-c", "import coverage"])
        if success:
            print_colored("Generating Python coverage report...", Colors.BLUE)
            
            # Run tests with coverage
            cmd = [
                "python3", "-m", "coverage", "run",
                "-m", "pytest", "tests/"
            ]
            success, stdout, stderr = run_command(cmd)
            
            if success:
                # Generate report
                success, stdout, stderr = run_command([
                    "python3", "-m", "coverage", "report", "--skip-empty"
                ])
                if success:
                    print_colored("✓ Python coverage report generated", Colors.GREEN)
                    print(stdout)
                else:
                    print_colored("✗ Failed to generate coverage report", Colors.RED)
            else:
                print_colored("✗ Failed to run coverage", Colors.RED)
        else:
            print_colored("Skipping Python coverage (coverage not installed)", Colors.YELLOW)
        
        os.chdir(original_cwd)
    
    # Go coverage (server & CLI)
    tunnel_dir = Path("tunnel")
    if tunnel_dir.exists():
        os.chdir(tunnel_dir)
        
        print_colored("Generating Go coverage report...", Colors.BLUE)
        
        # Run tests with coverage
        cmd = [
            "go", "test", "-coverprofile=coverage.out", "./..."
        ]
        success, stdout, stderr = run_command(cmd)
        
        if success:
            # Generate HTML report
            success, stdout, stderr = run_command([
                "go", "tool", "cover", "-html=coverage.out", "-o", "coverage.html"
            ])
            if success:
                print_colored("✓ Go coverage report generated (coverage.html)", Colors.GREEN)
                
                # Show coverage summary
                success, stdout, stderr = run_command([
                    "go", "tool", "cover", "-func=coverage.out"
                ])
                if success:
                    print(stdout)
            else:
                print_colored("✗ Failed to generate HTML coverage report", Colors.RED)
        else:
            print_colored("✗ Failed to run Go coverage", Colors.RED)
            print(stderr)
        
        os.chdir(original_cwd)


def main():
    parser = argparse.ArgumentParser(description="Run Portr tests")
    parser.add_argument("--component", choices=["admin", "server", "cli", "all"], 
                       default="all", help="Which component to test")
    parser.add_argument("--filter", help="Test filter pattern")
    parser.add_argument("--no-deps-check", action="store_true", 
                       help="Skip dependency checking")
    parser.add_argument("--lint", action="store_true", 
                       help="Run linting")
    parser.add_argument("--coverage", action="store_true", 
                       help="Generate coverage reports")
    parser.add_argument("--integration", action="store_true", 
                       help="Run integration tests")
    
    args = parser.parse_args()
    
    print_colored("Portr Test Runner", Colors.BOLD)
    print_colored("=" * 60, Colors.BLUE)
    
    # Check dependencies
    if not args.no_deps_check:
        if not check_dependencies():
            sys.exit(1)
    
    original_cwd = os.getcwd()
    results = []
    
    try:
        # Run tests based on component selection
        if args.component in ["admin", "all"]:
            os.chdir(original_cwd)
            success = run_admin_tests(args.filter)
            results.append(("Admin tests", success))
        
        if args.component in ["server", "all"]:
            os.chdir(original_cwd)
            success = run_server_tests(args.filter)
            results.append(("Server tests", success))
        
        if args.component in ["cli", "all"]:
            os.chdir(original_cwd)
            success = run_cli_tests(args.filter)
            results.append(("CLI tests", success))
        
        # Run integration tests if requested
        if args.integration:
            os.chdir(original_cwd)
            success = run_integration_tests()
            results.append(("Integration tests", success))
        
        # Run linting if requested
        if args.lint:
            os.chdir(original_cwd)
            success = run_linting()
            results.append(("Linting", success))
        
        # Generate coverage reports if requested
        if args.coverage:
            os.chdir(original_cwd)
            generate_coverage_report()
    
    finally:
        os.chdir(original_cwd)
    
    # Print summary
    print_header("Test Summary")
    
    overall_success = True
    for name, success in results:
        status = "✓ PASSED" if success else "✗ FAILED"
        color = Colors.GREEN if success else Colors.RED
        print_colored(f"{name:20} {status}", color)
        if not success:
            overall_success = False
    
    if overall_success:
        print_colored("\n🎉 All tests passed!", Colors.GREEN)
        sys.exit(0)
    else:
        print_colored("\n❌ Some tests failed!", Colors.RED)
        sys.exit(1)


if __name__ == "__main__":
    main()