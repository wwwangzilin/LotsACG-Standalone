"""
清理脚本：处理旧的 unknown 策略数据
"""
import asyncio
import aiosqlite
from pathlib import Path

DB_PATH = Path(__file__).parent / "data" / "pixiv_xp.db"

async def cleanup():
    async with aiosqlite.connect(DB_PATH) as db:
        # 1. 删除 strategy_stats 中的 unknown 条目
        cursor = await db.execute("DELETE FROM strategy_stats WHERE strategy = 'unknown'")
        stats_deleted = cursor.rowcount
        print(f"✅ 已删除 strategy_stats 中的 'unknown' 条目: {stats_deleted} 条")
        
        # 2. 将 push_history 中的 unknown 改为 legacy
        cursor = await db.execute("UPDATE push_history SET source = 'legacy' WHERE source = 'unknown' OR source IS NULL")
        history_updated = cursor.rowcount
        print(f"✅ 已将 push_history 中的 'unknown' 标记为 'legacy': {history_updated} 条")
        
        await db.commit()
        print("\n🎉 清理完成！MAB 策略统计将从零开始重新积累。")

if __name__ == "__main__":
    asyncio.run(cleanup())
