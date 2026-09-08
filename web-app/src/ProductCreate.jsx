import React from "react";
import { products } from "./data.js";
import { Modal, Field, Button } from "./components.jsx";
export default function ProductCreate({ act, close }) {
  return (
    <Modal title="发布官方商品" close={close}>
      <form
        className="form-stack"
        onSubmit={(e) => {
          e.preventDefault();
          const f = Object.fromEntries(new FormData(e.currentTarget));
          if (
            act(
              {
                type: "PRODUCT_CREATE",
                data: { ...f, stock: Number(f.stock) },
              },
              "官方商品已发布到演示商城",
            )
          )
            close();
        }}
      >
        <Field
          label="商品标题"
          name="title"
          required
          maxLength={60}
          placeholder="输入商品名称"
        />
        <Field
          label="商品简介"
          name="subtitle"
          required
          maxLength={120}
          placeholder="一句话介绍这件好物"
        />
        <div className="form-grid">
          <Field
            label="价格 / 信用点"
            name="price"
            type="number"
            step="0.01"
            min="0.01"
            max="9999999.99"
            required
          />
          <Field
            label="库存"
            name="stock"
            type="number"
            min="1"
            max="999"
            defaultValue="10"
            required
          />
        </div>
        <Field label="品牌">
          <select name="brand">
            <option>Apple</option>
            <option>NVIDIA</option>
            <option>AMD</option>
          </select>
        </Field>
        <Field label="现有商品素材">
          <select name="image">
            {products.map((p) => (
              <option key={p.id} value={p.image}>
                {p.title}
              </option>
            ))}
          </select>
        </Field>
        <div className="notice-box">
          使用 App
          的现成素材。正式商品图片上传、完整详情与受控发货模板将在后端联调时接入。
        </div>
        <Button type="submit">发布演示商品</Button>
      </form>
    </Modal>
  );
}
