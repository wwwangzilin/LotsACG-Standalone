import os
import sys
import subprocess
import time
import shutil

# 设置编码
sys.stdout.reconfigure(encoding='utf-8')

def clear_screen():
    os.system('cls' if os.name == 'nt' else 'clear')

def print_header(title):
    print("\n" + "=" * 40)
    print(f"   {title}")
    print("=" * 40 + "\n")

def run_command(cmd, shell=True, ignore_errors=False):
    try:
        if ignore_errors:
            subprocess.run(cmd, shell=shell, check=False, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        else:
            subprocess.run(cmd, shell=shell, check=True)
        return True
    except subprocess.CalledProcessError:
        return False

def check_env():
    print_header("[1/8] 环境检查")
    
    # Check Conda
    if shutil.which("conda"):
        print("   * 检测到 Conda")
        # Check if env exists
        result = subprocess.run("conda env list", shell=True, capture_output=True, text=True)
        if "pixiv-xp" not in result.stdout:
            print("   * 正在创建环境 pixiv-xp (可能需要几分钟)...")
            run_command("conda create -n pixiv-xp python=3.11 -y", ignore_errors=True)
        print("   * 注意：请确保在 Conda 环境中运行此脚本")
    else:
        print("   * 未检测到 Conda，使用系统 Python")
    
    print(f"   * Python 版本: {sys.version.split()[0]}")
    time.sleep(1)

def install_deps():
    print_header("[2/8] 安装依赖")
    print("   正在后台安装，请稍候...")
    run_command("pip install -r requirements.txt -q")
    print("   依赖安装完成")
    time.sleep(1)

def init_db():
    print_header("[3/8] 初始化数据库")
    run_command('python -c "import asyncio; from database import init_db; asyncio.run(init_db())"')
    print("   数据库已就绪")
    time.sleep(1)

def setup_token():
    print_header("[4/8] 获取 Pixiv Token")
    print("   Token 用于访问 Pixiv API 完整功能")
    print("   没有 Token 将以访客模式运行(功能受限)\n")
    
    choice = input("   是否获取 Token? (y/n): ").strip().lower()
    if choice == 'y':
        run_command("python get_token.py")

def setup_user_id():
    print_header("[5/8] 配置收藏分析目标")
    print("   系统会分析指定用户的公开收藏来构建 XP 画像")
    print("   输入 User ID (可在个人主页 URL 找到)\n")
    
    user_id = input("   请输入 User ID (直接回车跳过): ").strip()
    if user_id:
        update_config('user_id', user_id)
        print(f"   已保存 User ID: {user_id}")

def setup_schedule():
    print_header("[6/8] 定时任务设置")
    print("   设定每天自动运行的时间 (24小时制)")
    print("   例如: 12:30, 08:00, 23:59\n")
    
    t_input = input("   请输入每天推送时间 (默认为 12:00): ").strip()
    if t_input:
        try:
            t_input = t_input.replace("：", ":")
            h, m = map(int, t_input.split(":"))
            if 0 <= h < 24 and 0 <= m < 60:
                cron = f"{m} {h} * * *"
                update_config('cron', cron, section='scheduler')
                print(f"   已更新: 每天 {h:02d}:{m:02d} 执行")
            else:
                print("   ⚠️ 时间超出范围，未修改")
        except ValueError:
            print("   ⚠️ 格式错误，未修改")

def setup_ai():
    print_header("[7/8] AI 标签优化 (可选)")
    print("   使用 AI 过滤无意义标签、归类同义标签")
    print("   支持 OpenAI 及兼容 API (如 DeepSeek)\n")
    
    choice = input("   是否配置 AI? (y/n): ").strip().lower()
    if choice == 'y':
        api_key = input("   API Key: ").strip()
        base_url = input("   API Base URL (留空使用 OpenAI): ").strip()
        model = input("   模型名称 (默认 gpt-4o-mini): ").strip() or "gpt-4o-mini"
        
        update_config('enabled', 'true', section='ai')
        update_config('api_key', api_key, section='ai')
        if base_url:
            update_config('base_url', base_url, section='ai')
        update_config('model', model, section='ai')
        print("   AI 已配置")
    else:
        print("   已跳过 AI 配置")

def setup_notifier():
    print_header("[8/8] 配置推送方式")
    print("   1. Telegram Bot")
    print("   2. OneBot / QQ")
    print("   3. 跳过\n")
    
    choice = input("   请选择 (1/2/3): ").strip()
    
    if choice == '1':
        token = input("   Bot Token: ").strip()
        chat_id = input("   Chat ID: ").strip()
        update_config('type', 'telegram')
        update_config('bot_token', token)
        update_config('chat_id', chat_id)
        print("   Telegram 已配置")
        
    elif choice == '2':
        url = input("   WebSocket URL: ").strip()
        tid = input("   目标 QQ/群号: ").strip()
        type_choice = input("   类型 (1=私聊, 2=群聊): ").strip()
        ttype = "group" if type_choice == '2' else "private"
        
        update_config('type', 'onebot')
        update_config('ws_url', url)
        update_config('target_id', tid)
        update_config('target_type', ttype)
        print("   OneBot 已配置")

def update_config(key, value, section=None):
    """更新配置文件
    
    Args:
        key: 配置键名
        value: 配置值
        section: 所属段落（如 'ai' 表示 profiler.ai 下的键）
    """
    try:
        with open("config.yaml", "r", encoding="utf-8") as f:
            lines = f.readlines()
        
        new_lines = []
        in_section = False
        section_indent = 0
        
        for i, line in enumerate(lines):
            stripped = line.lstrip()
            current_indent = len(line) - len(stripped)
            
            if section:
                # 查找目标 section
                if stripped.startswith(f"{section}:"):
                    in_section = True
                    section_indent = current_indent
                    new_lines.append(line)
                    continue
                
                # 在 section 内查找 key
                if in_section:
                    # 检查是否离开了 section
                    if stripped and current_indent <= section_indent and not stripped.startswith(f"{key}:"):
                        in_section = False
                    elif stripped.startswith(f"{key}:"):
                        # 找到目标 key，替换值
                        indent = " " * current_indent
                        new_lines.append(f"{indent}{key}: {value}\n")
                        continue
                
                new_lines.append(line)
            else:
                # 简单替换
                if stripped.startswith(f"{key}:"):
                    indent = " " * current_indent
                    new_lines.append(f"{indent}{key}: {value}\n")
                else:
                    new_lines.append(line)
        
        with open("config.yaml", "w", encoding="utf-8") as f:
            f.writelines(new_lines)
    except Exception as e:
        print(f"   配置更新失败: {e}")

def main_menu():
    while True:
        clear_screen()
        print_header("Pixiv-XP-Pusher 主菜单")
        print("   1. 立即启动并常驻 (推荐)")
        print("   2. 仅启动定时任务")
        print("   3. 单次运行 (调试用)")
        print("   4. 启动网页管理")
        print("   5. 获取 Token")
        print("   6. 重新运行配置")
        print("   0. 退出\n")
        
        choice = input("   请选择: ").strip()
        
        if choice == '1':
            print("\n   🚀 正在立即启动任务，并在完成后转为后台常驻...")
            run_command("python main.py --now")
            input("\n   按回车键继续...")
            
        elif choice == '2':
            print("\n   ⏰ 启动定时调度器 (Ctrl+C 停止)")
            run_command("python main.py")
            input("\n   按回车键继续...")

        elif choice == '3':
            print("\n   🔧 执行单次推送调试...")
            run_command("python main.py --once")
            input("\n   按回车键继续...")
            
        elif choice == '4':
            print("\n   启动网页管理 (http://localhost:8000)")
            run_command("uvicorn web.app:app --host 0.0.0.0 --port 8000")
            input("\n   按回车键继续...")
            
        elif choice == '5':
            run_command("python get_token.py")
            input("\n   按回车键继续...")
            
        elif choice == '6':
            if os.path.exists(".initialized"):
                os.remove(".initialized")
            return  # restart wizard
            
        elif choice == '0':
            sys.exit(0)

def main():
    os.chdir(os.path.dirname(os.path.abspath(__file__)))
    
    if os.path.exists(".initialized"):
        main_menu()
    
    # Wizard
    clear_screen()
    print_header("首次运行向导")
    print("   欢迎使用 Pixiv-XP-Pusher")
    input("\n   按回车键开始配置...")
    
    check_env()
    install_deps()
    init_db()
    setup_token()
    setup_user_id()
    setup_schedule()
    setup_ai()
    setup_notifier()
    
    with open(".initialized", "w") as f:
        f.write("done")
        
    print("\n   配置完成！")
    time.sleep(1)
    main_menu()

if __name__ == "__main__":
    main()
